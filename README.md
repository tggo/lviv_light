## About

Use image from https://poweron.loe.lviv.ua parse it with OpenCV and convert to available hours range.

example:
```
Row 0: [green orange orange orange green green green green green green green green green orange orange orange orange green green orange orange green green orange]
Row 1: [green orange orange orange green green green green green green green green green orange orange orange orange green green green green green orange green]
Row 2: [green green green green orange green green green green green orange orange green green green green orange green green green green green orange orange]
Row 3: [green green green green green orange orange green green green orange orange orange green green green green orange orange green green green orange orange]
Row 4: [orange orange orange green green orange orange orange orange green green green green green green green green orange orange orange orange green green green]
Row 5: [orange green green green green orange orange orange orange green green green green green green green green orange orange orange orange green green green]
Available hours:  [0 4 5 6 7 8 9 10 11 12 17 18 19 20 21 23]
Available ranges:  [0-1 4-13 17-22 23-24]

```


## OpenCV installation
installation OpenCV on macOS M1 chip - https://gist.github.com/nucliweb/b2a234c673221af5ec24508da7d8b854

don't forget add:

```bash
-DOPENCV_GENERATE_PKGCONFIG=ON

pkg-config --cflags --libs opencv4
```
