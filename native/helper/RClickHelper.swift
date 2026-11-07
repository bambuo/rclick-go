import AppKit
import Foundation

private let outputLock = NSLock()

private func emit(_ object: [String: Any]) {
    guard let data = try? JSONSerialization.data(withJSONObject: object),
          let line = String(data: data, encoding: .utf8) else { return }
    outputLock.lock()
    FileHandle.standardOutput.write((line + "\n").data(using: .utf8)!)
    outputLock.unlock()
}

private func decodePayload(_ payload: Any?) -> String? {
    guard let payload else { return nil }
    if let text = payload as? String { return text }
    guard let data = try? JSONSerialization.data(withJSONObject: payload) else { return nil }
    return String(data: data, encoding: .utf8)
}

/// Native bridge for the Go main app. It owns the macOS APIs mygo does not
/// expose and forwards Finder notifications over stdin/stdout as JSON lines.
final class Helper: NSObject, NSApplicationDelegate {
    private var sharingService: NSSharingService?
    private var buffer = Data()

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.accessory)

        DistributedNotificationCenter.default().addObserver(
            forName: NSNotification.Name("RCLICK_FINDER_Main"),
            object: nil,
            queue: .main
        ) { note in
            let payload = note.object as? String ?? ""
            emit(["type": "message", "name": "RCLICK_FINDER_Main", "payload": payload])
        }

        FileHandle.standardInput.readabilityHandler = { [weak self] handle in
            let data = handle.availableData
            if data.isEmpty {
                handle.readabilityHandler = nil
                DispatchQueue.main.async {
                    NSApp.terminate(nil)
                }
                return
            }
            self?.consume(data)
        }

        emit(["type": "ready"])
    }

    private func consume(_ data: Data) {
        buffer.append(data)
        while let newline = buffer.firstIndex(of: 0x0A) {
            let lineData = buffer.subdata(in: buffer.startIndex..<newline)
            buffer.removeSubrange(buffer.startIndex...newline)
            guard !lineData.isEmpty,
                  let object = try? JSONSerialization.jsonObject(with: lineData) as? [String: Any] else { continue }
            DispatchQueue.main.async { [weak self] in
                self?.handle(object)
            }
        }
    }

    private func handle(_ command: [String: Any]) {
        switch command["cmd"] as? String ?? "" {
        case "send":
            let notification = command["name"] as? String ?? ""
            let payload = decodePayload(command["payload"]) ?? ""
            DistributedNotificationCenter.default().postNotificationName(
                NSNotification.Name(notification),
                object: payload,
                userInfo: nil,
                deliverImmediately: true
            )
        case "clipboard":
            let text = command["text"] as? String ?? ""
            let pasteboard = NSPasteboard.general
            pasteboard.clearContents()
            pasteboard.setString(text, forType: .string)
        case "open":
            for path in command["paths"] as? [String] ?? [] {
                NSWorkspace.shared.open(URL(fileURLWithPath: path))
            }
        case "openWith":
            openWith(command)
        case "airdrop":
            airDrop(command["paths"] as? [String] ?? [])
        case "trash":
            trash(command["paths"] as? [String] ?? [])
        case "hidden":
            setHidden(command["paths"] as? [String] ?? [], command["hidden"] as? Bool ?? true)
        case "confirm":
            confirm(command)
        case "quit":
            NSApp.terminate(nil)
        default:
            emit(["type": "error", "message": "unknown command"])
        }
    }

    private func confirm(_ command: [String: Any]) {
        let id = (command["id"] as? NSNumber)?.int64Value ?? 0
        let chinese = Locale.preferredLanguages.first?.hasPrefix("zh") ?? false
        let alert = NSAlert()
        alert.messageText = command["title"] as? String ?? (chinese ? "运行快捷命令？" : "Run Quick Command?")
        alert.informativeText = command["message"] as? String ?? ""
        alert.alertStyle = command["dangerous"] as? Bool == true ? .warning : .informational
        alert.addButton(withTitle: chinese ? "取消" : "Cancel")
        alert.addButton(withTitle: chinese ? "运行" : "Run")
        NSApp.activate(ignoringOtherApps: true)
        let response = alert.runModal()
        emit(["type": "confirmResult", "id": id, "confirmed": response == .alertSecondButtonReturn])
    }

    private func openWith(_ command: [String: Any]) {
        guard let appPath = command["appPath"] as? String else { return }
        let appURL = URL(fileURLWithPath: appPath)
        let configuration = NSWorkspace.OpenConfiguration()
        configuration.promptsUserIfNeeded = true
        configuration.arguments = command["arguments"] as? [String] ?? []
        if let environment = command["environment"] as? [String: String] {
            configuration.environment = environment
        }
        let urls = (command["paths"] as? [String] ?? []).map { URL(fileURLWithPath: $0) }
        guard !urls.isEmpty else { return }
        NSWorkspace.shared.open(urls, withApplicationAt: appURL, configuration: configuration) { _, error in
            if let error {
                emit(["type": "error", "message": "openWith: \(error.localizedDescription)"])
            }
        }
    }

    private func airDrop(_ paths: [String]) {
        let urls = paths.map { URL(fileURLWithPath: $0) }
        guard !urls.isEmpty, let service = NSSharingService(named: .sendViaAirDrop) else { return }
        sharingService = service
        service.perform(withItems: urls)
    }

    private func trash(_ paths: [String]) {
        for path in paths {
            do {
                try FileManager.default.trashItem(at: URL(fileURLWithPath: path), resultingItemURL: nil)
            } catch {
                emit(["type": "error", "message": "trash \(path): \(error.localizedDescription)"])
            }
        }
    }

    private func setHidden(_ paths: [String], _ hidden: Bool) {
        for path in paths {
            var url = URL(fileURLWithPath: path)
            do {
                var values = URLResourceValues()
                values.isHidden = hidden
                try url.setResourceValues(values)
            } catch {
                emit(["type": "error", "message": "hidden \(path): \(error.localizedDescription)"])
            }
        }
    }
}

@main
struct Launcher {
    static func main() {
        let application = NSApplication.shared
        let helper = Helper()
        application.delegate = helper
        application.run()
    }
}