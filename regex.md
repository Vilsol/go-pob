Here is a list of some regexes that seem to achieve pretty nice results going from Lua -> Go

Use these with caution and still check every time you apply one.

Make sure to use them with case sensitivity enabled!

| Find                        | Replace                  |
|-----------------------------|--------------------------|
| `local ([\w, ]+?) =`        | `$1 :=`                  |
| `([^\.])output\.(\w+?) `    | `$1actor.Output["$2"] `  |
| `([^\.])output\.(\w+?)$`    | `$1actor.Output["$2"]`   |
| `([^\.])output\.(\w+?)(\W)` | `$1actor.Output["$2"]$3` |
| `([^\.])output\[([^]]+)\]`  | `$1actor.Output[$2]`     |
| `(\W)then$`                 | `$1{`                    |
| `(\W)do$`                   | `$1{`                    |
| `^([ \t]+)elseif `          | `$1} else if `           |
| `^([ \t]+)else$`            | `$1} else {`             |
| `^([ \t]+)end$`             | `$1}`                    |
| `modDB\:(\w+)\(`            | `modDB.$1(`              |
| `Sum\("BASE",`              | `Sum(mod.TypeBase,`      |
| `Sum\("INC",`               | `Sum(mod.TypeIncrease,`  |
| `([^\.])\.\.([^\.])`        | `$1+$2`                  |
| `([ \t])--`                 | `$1//`                   |
| `m_max\(`                   | `max(`                   |
| `m_min\(`                   | `min(`                   |
| `m_floor\(`                 | `math.Floor(`            |
