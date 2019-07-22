```go
package main

import (
    "fmt"
    "log"

    "github.com/pm-esd/crontab"
)

func main() {

    ctab := crontab.New()

    err := ctab.AddJob("0 0 12 1 * *", myFunc)
    if err != nil {
        log.Println(err)
        return
    }
    ctab.MustAddJob("* * * * * *", myFunc)
    ctab.MustAddJob("0 0 12 * * *", myFunc3)

    ctab.MustAddJob("0 0 0 * * 1,2", myFunc2, "Monday and Tuesday midnight", 123)
    ctab.MustAddJob("0 */5 * * * *", myFunc2, "every five min", 0)

}

func myFunc() {
    fmt.Println("Helo, world")
}

func myFunc3() {
    fmt.Println("Noon!")
}

func myFunc2(s string, n int) {
    fmt.Println("We have params here, string", s, "and number", n)
}

```

## Crontab syntax

```

*    *     *     *     *     *
^    ^     ^     ^     ^     ^
|    |     |     |     |     |
|    |     |     |     |     +----- day of week (0-6) (Sunday=0)
|    |     |     |     +------- month (1-12)
|    |     |     +--------- day of month (1-31)
|    |     +----------- hour (0-23)
|    +------------- min (0-59)
+----------------second(0-59)
```

### Examples

+ `* * * * * *` run on every minute
+ `* 10 * * * *` run at 0:10, 1:10 etc
+ `* 10 15 * * *` run at 15:10 every day
+ `* * * 1 * *` run on every minute on 1st day of month
+ `0 0 0 1 1 *` Happy new year schedule
+ `0 0 0 * * 1` Run at midnight on every Monday

### Lists

+ `* * 10,15,19 * * *` run at 10:00, 15:00 and 19:00
+ `* 1-15 * * * *` run at 1, 2, 3...15 minute of each hour
+ `* 0 0-5,10 * * *` run on every hour from 0-5 and in 10 oclock

### Steps
+ `* */2 * * * *` run every two minutes
+ `* 10 */3 * * *` run every 3 hours on 10th min
+ `* 0 12 */2 * *` run at noon on every two days
+ `* 1-59/2 * * * *` run every two minutes, but on odd minutes
