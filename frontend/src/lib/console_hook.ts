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
  '37m': 'color:grey;',
  '38m': 'color:white;',
  '39m': 'color:unset;',

  '90m': 'color:gray;',
  '91m': 'color:#ff4a4a;',
  '92m': 'color:#52c052;',
  '93m': 'color:#fcfc69;',
  '94m': 'color:#5353ff;',
  '95m': 'color:#ff89ff;',
  '96m': 'color:#67ffff;',
  '97m': 'color:grey;',
  '98m': 'color:white;',
  '99m': 'color:unset;'
};

const oldLog = console.log;
console.log = (...args) => {
  if (args.length == 1 && typeof args[0] === 'string') {
    const allAnsiCodes = args[0].matchAll(ansiRegex);
    const cssMapped = [...allAnsiCodes].map((c) => cssMap[c[1]] || '');
    return oldLog(args[0].replaceAll(ansiRegex, '%c'), ...cssMapped);
  }
  return oldLog(...args);
};
