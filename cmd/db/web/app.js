const queryField = document.getElementById('sql');
const statusBox = document.getElementById('status');
const head = document.getElementById('thead');
const body = document.getElementById('tbody');

const exampleSQL = 'CREATE TABLE IF NOT EXISTS products (id INTEGER, name TEXT, price DECIMAL(10,2))';

async function runQuery() {
  const sql = queryField.value.trim();
  if (!sql) {
    statusBox.textContent = 'Enter a query first.';
    return;
  }

  statusBox.textContent = 'Executing...';
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

    renderRows(payload.rows || [], payload.columns || []);
    statusBox.textContent = `OK: ${payload.rows?.length ?? 0} row(s)`;
  } catch (err) {
    statusBox.textContent = err.message;
    head.innerHTML = '';
    body.innerHTML = '';
  }
}

function renderRows(rows, columns) {
  head.replaceChildren();
  body.replaceChildren();
  if (!rows.length) {
    const row = document.createElement('tr');
    const cell = document.createElement('td');
    cell.colSpan = Math.max(columns.length, 1);
    cell.textContent = 'No rows returned.';
    row.append(cell);
    body.append(row);
    return;
  }

  const resultColumns = columns.length ? columns : Object.keys(rows[0]);
  const headerRow = document.createElement('tr');
  for (const column of resultColumns) {
    const cell = document.createElement('th');
    cell.textContent = column;
    headerRow.append(cell);
  }
  head.append(headerRow);

  for (const result of rows) {
    const row = document.createElement('tr');
    for (const column of resultColumns) {
      const cell = document.createElement('td');
      cell.textContent = result[column] ?? '';
      row.append(cell);
    }
    body.append(row);
  }
}

document.getElementById('run').addEventListener('click', runQuery);
document.getElementById('example').addEventListener('click', () => {
  queryField.value = exampleSQL;
  runQuery();
});
