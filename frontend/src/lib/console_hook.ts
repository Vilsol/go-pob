const ansiRegex = new RegExp(
  [
    '[\\u001B\\u009B][[\\]()#;?]*(?:(?:(?:(?:;[-a-zA-Z\\d\\/#&.:=?%@~_]+)*|[a-zA-Z\\d]+(?:;[-a-zA-Z\\d\\/#&.:=?%@~_]*)*)?\\u0007)',
    '((?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PR-TZcf-nq-uy=><~]))'
  ].join('|'),
  'g'
);

const cssMap: Record<string, string> = {
  '0m': 'font-weight:unset;color:unset;',
  '1m': 'font-weight:bold;',
  '30m': 'color:black;',
  '31m': 'color:red;',
  '32m': 'color:green;',
  '33m': 'color:yellow;',
  '34m': 'color:blue;',
  '35m': 'color:magenta;',
  '36m': 'color:cyan;',
  '37m': 'color:white;',
  '39m': 'color:unset;'
};

const oldLog = console.log;
console.log = (...args) => {
  if (args.length == 1) {
    const allAnsiCodes = (args[0] as string).matchAll(ansiRegex);
    const cssMapped = [...allAnsiCodes].map((c) => cssMap[c[1] as string] || '');
    return oldLog(args[0].replaceAll(ansiRegex, '%c'), ...cssMapped);
  }
  return oldLog(...args);
};
