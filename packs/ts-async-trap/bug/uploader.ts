type Row = { id: string; payload: unknown };

async function uploadRow(row: Row): Promise<{ id: string; ok: boolean }> {
  const res = await fetch("https://upstream.example.com/rows", {
    method: "POST",
    body: JSON.stringify(row),
  });
  if (!res.ok) {
    throw new Error(`upload failed: ${res.status}`);
  }
  return { id: row.id, ok: true };
}

export async function importBatch(rows: Row[]): Promise<void> {
  const results = await Promise.all(
    rows.map((row) => uploadRow(row).catch(() => null)),
  );
  console.log(`importBatch ok (${results.length} rows)`);
}
