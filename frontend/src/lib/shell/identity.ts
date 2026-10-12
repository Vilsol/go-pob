const ids = new WeakMap<object, number>();
let next = 0;

export const identityOf = (value: object | undefined | null): number => {
  if (!value) return 0;
  let id = ids.get(value);
  if (id === undefined) {
    id = ++next;
    ids.set(value, id);
  }
  return id;
};
