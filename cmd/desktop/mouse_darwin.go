//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework WebKit

#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>

// Callback from Go
extern void onMouseButton(int button);

static id mouseMonitor = nil;

static void startMouseMonitor() {
	// Monitor otherMouseDown for standard mice that send button 3 (back) / 4 (forward).
	// Note: Logitech Options+ intercepts these at the driver level and sends them as
	// NX_SYSDEFINED system events instead. For Logitech mice, users should configure
	// back/forward buttons to send Cmd+[ / Cmd+] keyboard shortcuts in Logi Options+,
	// which are handled by the app's View menu.
	mouseMonitor = [NSEvent addLocalMonitorForEventsMatchingMask:NSEventMaskOtherMouseDown
		handler:^NSEvent*(NSEvent* event) {
			int btn = (int)[event buttonNumber];
			if (btn == 3 || btn == 4) {
				onMouseButton(btn);
				return nil; // consume the event — don't pass to WebView
			}
			return event;
		}];
}

static void stopMouseMonitor() {
	if (mouseMonitor != nil) {
		[NSEvent removeMonitor:mouseMonitor];
		mouseMonitor = nil;
	}
}

// Navigate using WKWebView's native goBack/goForward on the main thread.
// This avoids crashes from evaluateJavaScript racing with WebView internal state.
static void navigateBack() {
	dispatch_async(dispatch_get_main_queue(), ^{
		NSWindow *window = [[NSApplication sharedApplication] mainWindow];
		if (!window) return;
		// Find the WKWebView in the window's view hierarchy
		NSView *contentView = [window contentView];
		for (NSView *subview in [contentView subviews]) {
			if ([subview isKindOfClass:[WKWebView class]]) {
				WKWebView *webView = (WKWebView *)subview;
				if ([webView canGoBack]) {
					[webView goBack];
				}
				return;
			}
			// Check one level deeper (Wails wraps in a container)
			for (NSView *nested in [subview subviews]) {
				if ([nested isKindOfClass:[WKWebView class]]) {
					WKWebView *webView = (WKWebView *)nested;
					if ([webView canGoBack]) {
						[webView goBack];
					}
					return;
				}
			}
		}
	});
}

static void navigateForward() {
	dispatch_async(dispatch_get_main_queue(), ^{
		NSWindow *window = [[NSApplication sharedApplication] mainWindow];
		if (!window) return;
		NSView *contentView = [window contentView];
		for (NSView *subview in [contentView subviews]) {
			if ([subview isKindOfClass:[WKWebView class]]) {
				WKWebView *webView = (WKWebView *)subview;
				if ([webView canGoForward]) {
					[webView goForward];
				}
				return;
			}
			for (NSView *nested in [subview subviews]) {
				if ([nested isKindOfClass:[WKWebView class]]) {
					WKWebView *webView = (WKWebView *)nested;
					if ([webView canGoForward]) {
						[webView goForward];
					}
					return;
				}
			}
		}
	});
}
*/
import "C"

import "context"

// appCtx kept for potential future use but no longer needed for navigation.
var appCtx context.Context

//export onMouseButton
func onMouseButton(button C.int) {
	switch int(button) {
	case 3: // back
		C.navigateBack()
	case 4: // forward
		C.navigateForward()
	}
}

func startNativeMouseMonitor(ctx context.Context) {
	appCtx = ctx
	C.startMouseMonitor()
}

func stopNativeMouseMonitor() {
	C.stopMouseMonitor()
}
