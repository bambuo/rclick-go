//
//  SharedConfigStore.swift
//  FinderSyncExt
//

import Foundation
import OSLog

struct SharedMenuSettings: Decodable {
    var showForItems = true
    var showForContainer = true
    var showForSidebar = true
    var showToolbarItem = true
    var showSubMenuForApplication = false
    var showSubMenuForAction = false
    var copySeparator = ""

    init() {}

    private enum CodingKeys: String, CodingKey {
        case showForItems
        case showForContainer
        case showForSidebar
        case showToolbarItem
        case showSubMenuForApplication
        case showSubMenuForAction
        case copySeparator
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        showForItems = try container.decodeIfPresent(Bool.self, forKey: .showForItems) ?? true
        showForContainer = try container.decodeIfPresent(Bool.self, forKey: .showForContainer) ?? true
        showForSidebar = try container.decodeIfPresent(Bool.self, forKey: .showForSidebar) ?? true
        showToolbarItem = try container.decodeIfPresent(Bool.self, forKey: .showToolbarItem) ?? true
        showSubMenuForApplication = try container.decodeIfPresent(Bool.self, forKey: .showSubMenuForApplication) ?? false
        showSubMenuForAction = try container.decodeIfPresent(Bool.self, forKey: .showSubMenuForAction) ?? false
        copySeparator = try container.decodeIfPresent(String.self, forKey: .copySeparator) ?? ""
    }
}

struct SharedPermissionDir: Decodable {
    var id: String
    var url: String
    var bookmark: String?
}

struct SharedConfig: Decodable {
    var schemaVersion: Int
    var revision: Int64
    var apps: [OpenWithApp]
    var actions: [RCAction]
    var newFiles: [NewFile]
    var commonDirs: [CommonDir]
    var permissionDirs: [SharedPermissionDir]
    var quickCommands: [QuickCommand]
    var menu: SharedMenuSettings

    static let empty = SharedConfig(
        schemaVersion: 1,
        revision: 0,
        apps: [],
        actions: [],
        newFiles: [],
        commonDirs: [],
        permissionDirs: [],
        quickCommands: [],
        menu: SharedMenuSettings()
    )

    init(
        schemaVersion: Int,
        revision: Int64,
        apps: [OpenWithApp],
        actions: [RCAction],
        newFiles: [NewFile],
        commonDirs: [CommonDir],
        permissionDirs: [SharedPermissionDir],
        quickCommands: [QuickCommand],
        menu: SharedMenuSettings
    ) {
        self.schemaVersion = schemaVersion
        self.revision = revision
        self.apps = apps
        self.actions = actions
        self.newFiles = newFiles
        self.commonDirs = commonDirs
        self.permissionDirs = permissionDirs
        self.quickCommands = quickCommands
        self.menu = menu
    }

    private enum CodingKeys: String, CodingKey {
        case schemaVersion
        case revision
        case apps
        case actions
        case newFiles
        case commonDirs
        case permissionDirs
        case quickCommands
        case menu
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        schemaVersion = try container.decodeIfPresent(Int.self, forKey: .schemaVersion) ?? 1
        revision = try container.decodeIfPresent(Int64.self, forKey: .revision) ?? 0
        apps = try container.decodeIfPresent([OpenWithApp].self, forKey: .apps) ?? []
        actions = try container.decodeIfPresent([RCAction].self, forKey: .actions) ?? []
        newFiles = try container.decodeIfPresent([NewFile].self, forKey: .newFiles) ?? []
        commonDirs = try container.decodeIfPresent([CommonDir].self, forKey: .commonDirs) ?? []
        permissionDirs = try container.decodeIfPresent([SharedPermissionDir].self, forKey: .permissionDirs) ?? []
        quickCommands = try container.decodeIfPresent([QuickCommand].self, forKey: .quickCommands) ?? []
        menu = try container.decodeIfPresent(SharedMenuSettings.self, forKey: .menu) ?? SharedMenuSettings()
    }

    var permissionURLs: [URL] {
        permissionDirs
            .map { $0.url }
            .filter { !$0.isEmpty }
            .map { URL(fileURLWithPath: $0) }
    }
}

@MainActor
final class SharedConfigStore {
    static let shared = SharedConfigStore()

    private static let logger = Logger(
        subsystem: Bundle.main.bundleIdentifier ?? "RClick",
        category: "SharedConfig"
    )

    private struct FileStamp: Equatable {
        var modified: Date
        var size: UInt64

        init?(url: URL) {
            guard let attributes = try? FileManager.default.attributesOfItem(atPath: url.path),
                  let modified = attributes[.modificationDate] as? Date,
                  let size = attributes[.size] as? NSNumber
            else {
                return nil
            }
            self.modified = modified
            self.size = size.uint64Value
        }
    }

    private let configURL: URL
    private var cached = SharedConfig.empty
    private var cachedStamp: FileStamp?

    init(configURL: URL? = nil) {
        self.configURL = configURL ?? Self.defaultURL()
    }

    static func defaultURL() -> URL {
        // A sandboxed appex sees its container as the home directory;
        // PathSecurityChecker trims it to the real user home.
        URL(fileURLWithPath: PathSecurityChecker.getRealHomeDir())
            .appendingPathComponent("Library/Application Support/RClick/config.json")
    }

    var current: SharedConfig {
        cached
    }

    @discardableResult
    func reloadIfNeeded() -> SharedConfig {
        let stamp = FileStamp(url: configURL)
        guard stamp != cachedStamp else {
            return cached
        }
        return load(stamp: stamp)
    }

    @discardableResult
    func reload() -> SharedConfig {
        load(stamp: FileStamp(url: configURL))
    }

    private func load(stamp: FileStamp?) -> SharedConfig {
        guard let stamp else {
            Self.logger.warning("config.json not found at \(self.configURL.path, privacy: .public)")
            cached = .empty
            cachedStamp = nil
            return cached
        }

        do {
            let data = try Data(contentsOf: configURL)
            let decoded = try JSONDecoder().decode(SharedConfig.self, from: data)
            cached = decoded
            cachedStamp = stamp
            Self.logger.info("loaded config revision \(decoded.revision)")
        } catch {
            cachedStamp = stamp
            Self.logger.error("failed to decode config: \(error.localizedDescription, privacy: .public)")
        }
        return cached
    }
}
