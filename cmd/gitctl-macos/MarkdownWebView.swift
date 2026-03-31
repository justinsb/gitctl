import SwiftUI
import WebKit

// MARK: - URL-based Detail Web View

/// A WKWebView that loads a URL from the backend and allows in-app navigation
/// (tab switching, form submissions) while opening external links in the system browser.

private let backendHost = "localhost"
private let backendPort = 8484

#if os(macOS)
struct DetailWebView: NSViewRepresentable {
    let url: URL

    func makeCoordinator() -> Coordinator {
        Coordinator()
    }

    func makeNSView(context: Context) -> WKWebView {
        let webView = WKWebView(frame: .zero)
        webView.navigationDelegate = context.coordinator
        webView.load(URLRequest(url: url))
        context.coordinator.currentURL = url
        return webView
    }

    func updateNSView(_ webView: WKWebView, context: Context) {
        if context.coordinator.currentURL != url {
            context.coordinator.currentURL = url
            webView.load(URLRequest(url: url))
        }
    }

    class Coordinator: NSObject, WKNavigationDelegate {
        var currentURL: URL?

        func webView(_ webView: WKWebView, decidePolicyFor navigationAction: WKNavigationAction, decisionHandler: @escaping (WKNavigationActionPolicy) -> Void) {
            guard let requestURL = navigationAction.request.url else {
                decisionHandler(.allow)
                return
            }

            // Allow internal navigation (same backend host) for tab switching and form POSTs.
            if isBackendURL(requestURL) {
                decisionHandler(.allow)
                return
            }

            // Open external links in system browser.
            if navigationAction.navigationType == .linkActivated {
                NSWorkspace.shared.open(requestURL)
                decisionHandler(.cancel)
            } else {
                decisionHandler(.allow)
            }
        }
    }
}
#else
struct DetailWebView: UIViewRepresentable {
    let url: URL

    func makeCoordinator() -> Coordinator {
        Coordinator()
    }

    func makeUIView(context: Context) -> WKWebView {
        let webView = WKWebView(frame: .zero)
        webView.navigationDelegate = context.coordinator
        webView.load(URLRequest(url: url))
        context.coordinator.currentURL = url
        return webView
    }

    func updateUIView(_ webView: WKWebView, context: Context) {
        if context.coordinator.currentURL != url {
            context.coordinator.currentURL = url
            webView.load(URLRequest(url: url))
        }
    }

    class Coordinator: NSObject, WKNavigationDelegate {
        var currentURL: URL?

        func webView(_ webView: WKWebView, decidePolicyFor navigationAction: WKNavigationAction, decisionHandler: @escaping (WKNavigationActionPolicy) -> Void) {
            guard let requestURL = navigationAction.request.url else {
                decisionHandler(.allow)
                return
            }

            // Allow internal navigation (same backend host) for tab switching and form POSTs.
            if isBackendURL(requestURL) {
                decisionHandler(.allow)
                return
            }

            // Open external links in system browser.
            if navigationAction.navigationType == .linkActivated {
                UIApplication.shared.open(requestURL)
                decisionHandler(.cancel)
            } else {
                decisionHandler(.allow)
            }
        }
    }
}
#endif

/// Checks if a URL points to our local backend.
private func isBackendURL(_ url: URL) -> Bool {
    return url.host == backendHost && url.port == backendPort
}
