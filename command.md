# Redis Server Assisted Client Side Caching

## default tracking

```shell
# client 1
> SET a 100
```

```shell
# client 3
> CLIENT ID
12
> SUBSCRIBE __redis__:invalidate
1) "subscribe"
2) "__redis__:invalidate"
3) (integer) 1
```

```shell
# client 2
> CLIENT TRACKING ON REDIRECT 12
> GET a # tracking
```

```shell
# client 1
> SET a 200
```

```shell
# client 3
1) "message"
2) "__redis__:invalidate"
3) 1) "a"
```

## broadcasting tracking

```shell
# client 3
> CLIENT ID
12
> SUBSCRIBE __redis__:invalidate
1) "subscribe"
2) "__redis__:invalidate"
3) (integer) 1
```

```shell
# client 2
CLIENT TRACKING ON BCAST PREFIX cache: REDIRECT 12
```

```shell
# client 1
> SET cache:name "Alice"
> SET cache:age 26
```

```shell
# client 3
1) "message"
2) "__redis__:invalidate"
3) 1) "cache:name"
1) "message"
2) "__redis__:invalidate"
3) 1) "cache:age"
```
