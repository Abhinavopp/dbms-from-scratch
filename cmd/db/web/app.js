const queryField = document.getElementById('sql');
const statusBox = document.getElementById('status');
const head = document.getElementById('thead');
const body = document.getElementById('tbody');
const schemaTree = document.getElementById('schemaTree');
const emptyState = document.getElementById('emptyState');

const sampleQueries = [
  "CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)",
  "INSERT INTO users (id, name, age) VALUES (1, 'Ana', 21)",
  "SELECT * FROM users LIMIT 10;",
  "SELECT name, age + 1 AS next_age FROM users INDEX BY age > 18 FILTER age < 30 LIMIT 10;"
];

function setStatus(message, kind = 'neutral') {
  statusBox.textContent = message;
  statusBox.className = `status ${kind}`;
}

function normalizeTableName(name) {
  return String(name || '').trim();
}

function renderSchema(tables) {
  schemaTree.innerHTML = '';

  const root = document.createElement('ul');
  root.className = 'tree-table';

  if (!tables || !tables.length) {
    const empty = document.createElement('li');
    empty.textContent = 'No tables created yet';
    root.appendChild(empty);
    schemaTree.appendChild(root);
    return;
  }

  for (const table of tables) {
    const tableRow = document.createElement('li');
    const label = document.createElement('div');
    label.className = 'tree-item';
    label.innerHTML = '<span class="tree-bullet">▸</span><span class="tree-name">' + escapeHtml(table.name) + '</span>';
    tableRow.appendChild(label);

    const columns = document.createElement('ul');
    columns.className = 'tree-table tree-indent';

    const columnsItem = document.createElement('li');
    columnsItem.innerHTML = '<span class="tree-bullet">▸</span> Columns';
    columns.appendChild(columnsItem);

    for (const column of table.columns || []) {
      const col = document.createElement('li');
      col.innerHTML = '<span class="tree-bullet">•</span> ' + escapeHtml(column.name) + ' ' + escapeHtml(column.type);
      columns.appendChild(col);
    }

    if (table.primaryKey) {
      const primary = document.createElement('li');
      primary.innerHTML = '<span class="tree-bullet">▸</span> Primary Key: ' + escapeHtml(table.primaryKey);
      columns.appendChild(primary);
    }

    if ((table.indexes || []).length) {
      const indexes = document.createElement('li');
      indexes.innerHTML = '<span class="tree-bullet">▸</span> Indexes';
      columns.appendChild(indexes);
      for (const index of table.indexes) {
        const idx = document.createElement('li');
        idx.innerHTML = '<span class="tree-bullet">•</span> ' + escapeHtml(index);
        columns.appendChild(idx);
      }
    }

    tableRow.appendChild(columns);
    root.appendChild(tableRow);
  }

  schemaTree.appendChild(root);
}

function escapeHtml(value) {
  return String(value)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
}

function loadSchema() {
  fetch('/api/schema')
    .then((response) => response.json())
    .then((payload) => renderSchema(payload.tables || []))
    .catch(() => renderSchema([]));
}

function renderRows(rows, columns) {
  head.replaceChildren();
  body.replaceChildren();
  emptyState.classList.toggle('hidden', rows.length > 0 || columns.length > 0);

  if (!rows.length) {
    if (!columns.length) {
      const row = document.createElement('tr');
      const cell = document.createElement('td');
      cell.colSpan = 1;
      cell.textContent = 'No rows returned.';
      row.appendChild(cell);
      body.appendChild(row);
    } else {
      const headerRow = document.createElement('tr');
      for (const column of columns) {
        const cell = document.createElement('th');
        cell.textContent = column;
        headerRow.appendChild(cell);
      }
      head.appendChild(headerRow);
      const emptyRow = document.createElement('tr');
      const plainCell = document.createElement('td');
      plainCell.colSpan = columns.length || 1;
      plainCell.textContent = 'No rows returned for this query.';
      emptyRow.appendChild(plainCell);
      body.appendChild(emptyRow);
    }
    return;
  }

  const resultColumns = columns.length ? columns : Object.keys(rows[0]);
  const headerRow = document.createElement('tr');
  for (const column of resultColumns) {
    const cell = document.createElement('th');
    cell.textContent = column;
    headerRow.appendChild(cell);
  }
  head.appendChild(headerRow);

  for (const result of rows) {
    const row = document.createElement('tr');
    for (const column of resultColumns) {
      const cell = document.createElement('td');
      cell.textContent = result[column] ?? '';
      row.appendChild(cell);
    }
    body.appendChild(row);
  }
}

async function runQuery() {
  const sql = queryField.value.trim();
  if (!sql) {
    setStatus('Enter a query first.', 'error');
    return;
  }

  setStatus('Running...', 'running');

  try {
    const response = await fetch('/api/query', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ sql })
    });

    const payload = await response.json();
    if (!response.ok) {
      throw new Error(payload.error || 'Query failed');
    }

    const resultColumns = Array.isArray(payload.columns) ? payload.columns : [];
    const resultRows = Array.isArray(payload.rows) ? payload.rows : [];
    renderRows(resultRows, resultColumns);
    setStatus('✓ Query executed successfully', 'success');
    loadSchema();
  } catch (err) {
    renderRows([], []);
    setStatus('✕ Query failed', 'error');
    console.error(err);
  }
}

function formatSQL() {
  const sql = queryField.value.trim();
  if (!sql) {
    return;
  }

  const simplified = sql
    .replace(/\s+/g, ' ')
    .replace(/\s*\(\s*/g, ' ( ')
    .replace(/\s*\)\s*/g, ' ) ')
    .replace(/\s*,\s*/g, ', ')
    .replace(/\s*;\s*$/g, ';')
    .trim();

  queryField.value = simplified;
}

function clearEditor() {
  queryField.value = '';
  renderRows([], []);
  setStatus('Ready', 'neutral');
}

queryField.addEventListener('keydown', (event) => {
  if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') {
    event.preventDefault();
    runQuery();
  }
});

document.getElementById('run').addEventListener('click', runQuery);
document.getElementById('format').addEventListener('click', formatSQL);
document.getElementById('clear').addEventListener('click', clearEditor);

(function init() {
  loadSchema();
  queryField.value = sampleQueries[0];
  renderRows([], []);
  setStatus('Ready', 'neutral');
})();
