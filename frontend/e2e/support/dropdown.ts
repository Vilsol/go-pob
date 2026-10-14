import { expect, type Locator } from '@playwright/test';

const listOf = async (box: Locator) => box.page().locator(`[id="${await box.getAttribute('aria-controls')}"]`);
const open = async (box: Locator) => {
  if ((await box.getAttribute('aria-expanded')) !== 'true') await box.click();
  const list = await listOf(box);
  await expect(list).toBeVisible();
  return list;
};

export const entries = async (box: Locator): Promise<string[]> => {
  const list = await open(box);
  const out = (await list.getByRole('option').allTextContents()).map((t) => t.trim());
  await box.click();
  return out;
};

export const expectEntries = (box: Locator, want: string[], timeout = 30_000) => expect.poll(() => entries(box), { timeout }).toEqual(want);

export const choose = async (box: Locator, label: string) => {
  const list = await open(box);
  await list.getByRole('option', { name: label, exact: true }).first().click();
  await expect(box).toHaveAttribute('aria-expanded', 'false');
};
