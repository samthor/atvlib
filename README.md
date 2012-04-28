atvlib
======

Small library for interacting with an Apple TV on a local network. Requires Go 1+.

As a demo app, this includes a tiny binary which can be used to play something that the Apple TV might understand, such as a H.264 .mp4 file.

To install the binary, run the following.

    go install github.com/samthor/atvlib/atvplay

`atvplay` will now be available in your `$GOBIN` (or `$GOPATH/bin`, or...).

