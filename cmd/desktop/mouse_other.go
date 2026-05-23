//go:build !darwin

package main

import "context"

func startNativeMouseMonitor(ctx context.Context) {}
func stopNativeMouseMonitor()                     {}
