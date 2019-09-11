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

# represent topomap:
<img src="https://github.com/uxff/gravity_sim_go/raw/master/image/topomap-20180902070444.png">

- ridge appear
- highland cover

