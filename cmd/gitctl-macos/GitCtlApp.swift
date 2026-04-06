import SwiftUI

#if os(macOS)
class AppDelegate: NSObject, NSApplicationDelegate {
    var backendManager: BackendManager?

    func applicationWillTerminate(_ notification: Notification) {
        backendManager?.stop()
    }
}
#endif

@main
struct GitCtlApp: App {
    #if os(macOS)
    @NSApplicationDelegateAdaptor(AppDelegate.self) var appDelegate
    #endif

    @StateObject private var backendManager = BackendManager()

    var body: some Scene {
        WindowGroup {
            ContentView()
                .onAppear {
                    #if os(macOS)
                    appDelegate.backendManager = backendManager
                    #endif
                    backendManager.start()
                }
        }
    }
}
