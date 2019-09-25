# topograph simulate

To simulate a topograph maker, with moutains, with continent, with sea, with river, with rain.

Make a random continent, give it some moutains, then put rain/water on it, simulate water flow on it.

Now, simulating is not success yet.

useage:
```
$ go build apps/appv3/topomaker.go
$ ./topomaker --zoom 3 -h 500 -w 500 --hill 30 --hill-wide 30 --ridge 10 --ridge-len 50 --ridge-wide 40 --dropnum 100 --times 1000
```

todo: use updater

# represent topomap (4P):
<img src="https://github.com/uxff/topograph-maker/raw/master/image/topomap-20190925215426.png">
<img src="https://github.com/uxff/topograph-maker/raw/master/image/topomap-20190924224007.png">
<img src="https://github.com/uxff/topograph-maker/raw/master/image/topomap-20190924223926.png">
<img src="https://github.com/uxff/topograph-maker/raw/master/image/topomap-20190924223543.png">

- ridge appear
- highland cover

# sundries 
batch rename
```
 for i in `ls output/ `; do echo mv  $i topomap${i#*map}; mv output/$i  output/topomap${i#*map}; done
```
