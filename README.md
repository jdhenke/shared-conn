# Shared Conn

> Passing a `net.Conn` between processes.

## Intro

Toy program meant to address this [go forum issue](https://forum.golangbridge.org/t/bind-address-already-in-use-even-after-listener-closed/1510).

This program creates a listener by binding to the supplied address, accesses the
underlying FD, and passes it to a child process.

The child process recreates the listener using the shared FD, then attempts to
rebind the same address as the parent.

It exposes two issues via command line flags.

## Setup

```
go get github.com/jdhenke/shared-conn
```

## Usage

```
$ shared-conn -h
Usage of shared-conn:
  -addr string
    	address on which to listen (default ":1234")
  -ignoreFiles
    	ignore closing FDs
  -triggerRace
    	trigger race condition
```

## Example

Running with default settings *should* pass, although a race exists.

```
$ shared-conn
parent | 17:36:02.190679 Using address=`:1234`
parent | 17:36:02.193514 Closed listener.
parent | 17:36:02.193528 Closed file.
child  | 17:36:02.196729 Using address=`:1234`
child  | 17:36:02.196920 Closed listener.
child  | 17:36:02.196929 Closed file.
child  | 17:36:02.196931 Rebinding...
child  | 17:36:02.196983 PASS
parent | 17:36:02.197860 PASS
```

Running with trigger race *should* fail, although a race exists.

```
$ shared-conn -triggerRace
parent | 17:36:08.220841 Using address=`:1234`
child  | 17:36:08.226717 Using address=`:1234`
child  | 17:36:08.226917 Closed listener.
child  | 17:36:08.226921 Closed file.
child  | 17:36:08.226922 Rebinding...
child  | 17:36:08.226972 Child could not recreate listener: listen tcp :1234: bind: address already in use
parent | 17:36:10.226090 Closed listener.
parent | 17:36:10.226135 Closed file.
parent | 17:36:10.226188 Child process error: exit status 1
```

Leaving the files open then attempting to rebind should always fail.

```
$ shared-conn -ignoreFiles
parent | 17:36:14.632811 Using address=`:1234`
parent | 17:36:14.635907 Closed listener.
child  | 17:36:14.638920 Using address=`:1234`
child  | 17:36:14.639092 Closed listener.
child  | 17:36:14.639095 Rebinding...
child  | 17:36:14.639142 Child could not recreate listener: listen tcp :1234: bind: address already in use
parent | 17:36:14.639982 Child process error: exit status 1
```
