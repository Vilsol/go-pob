Here is a list of some regexes that seem to achieve pretty nice results going from Lua -> Go

Use these with caution and still check every time you apply one.

Make sure to use them with case sensitivity enabled!

| Find                 | Replace                 |
|----------------------|-------------------------|
| `local (\w+) =`      | `$1 :=`                 |
| `output\.(\w+?) `    | `actor.Output["$1"] `   |
| `output\.(\w+?)$`    | `actor.Output["$1"]`    |
| `output\.(\w+?)(\W)` | `actor.Output["$1"]$2`  |
| `output\[([^]]+)\]`  | `actor.Output[$1]`      |
| `then$`              | `{`                     |
| `do$`                | `{`                     |
| `^(\s+)elseif `      | `$1} else if `          |
| `^(\s+)else$`        | `$1} else {`            |
| `^(\s+)end$`         | `$1}`                   |
| `modDB\:(\w+)\(`     | `modDB.$1(`             |
| `Sum\("BASE",`       | `Sum(mod.TypeBase,`     |
| `Sum\("INC",`        | `Sum(mod.TypeIncrease,` |
| `\.\.`               | `+`                     |
