export namespace deeplink {
	
	export class Target {
	    Type: string;
	    ID: string;
	
	    static createFrom(source: any = {}) {
	        return new Target(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Type = source["Type"];
	        this.ID = source["ID"];
	    }
	}

}

export namespace types {
	
	export class AppConfig {
	    metroMakerDataPath?: string;
	    executablePath?: string;
	    githubToken?: string;
	    checkForUpdatesOnLaunch: boolean;
	    setupCompleted: boolean;
	    chromeSandboxPath?: string;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.metroMakerDataPath = source["metroMakerDataPath"];
	        this.executablePath = source["executablePath"];
	        this.githubToken = source["githubToken"];
	        this.checkForUpdatesOnLaunch = source["checkForUpdatesOnLaunch"];
	        this.setupCompleted = source["setupCompleted"];
	        this.chromeSandboxPath = source["chromeSandboxPath"];
	    }
	}
	export class AppVersionResponse {
	    status: string;
	    message: string;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new AppVersionResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.version = source["version"];
	    }
	}
	export class AssetDownloadCountsResponse {
	    status: string;
	    message: string;
	    assetType: string;
	    assetId: string;
	    counts: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new AssetDownloadCountsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.assetType = source["assetType"];
	        this.assetId = source["assetId"];
	        this.counts = source["counts"];
	    }
	}
	export class ConfigData {
	    name: string;
	    code: string;
	    description: string;
	    population: number;
	    country?: string;
	    thumbnailBbox?: number[];
	    bbox?: number[];
	    creator: string;
	    version: string;
	    // Go type: struct { Latitude float64 "json:\"latitude\""; Longitude float64 "json:\"longitude\""; Zoom float64 "json:\"zoom\""; Pitch *float64 "json:\"pitch,omitempty\""; Bearing float64 "json:\"bearing\"" }
	    initialViewState: any;
	
	    static createFrom(source: any = {}) {
	        return new ConfigData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.code = source["code"];
	        this.description = source["description"];
	        this.population = source["population"];
	        this.country = source["country"];
	        this.thumbnailBbox = source["thumbnailBbox"];
	        this.bbox = source["bbox"];
	        this.creator = source["creator"];
	        this.version = source["version"];
	        this.initialViewState = this.convertValues(source["initialViewState"], Object);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AssetInstallResponse {
	    status: string;
	    message: string;
	    assetType: string;
	    assetId: string;
	    version: string;
	    config?: ConfigData;
	    errorType?: string;
	
	    static createFrom(source: any = {}) {
	        return new AssetInstallResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.assetType = source["assetType"];
	        this.assetId = source["assetId"];
	        this.version = source["version"];
	        this.config = this.convertValues(source["config"], ConfigData);
	        this.errorType = source["errorType"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AssetUninstallResponse {
	    status: string;
	    message: string;
	    assetType: string;
	    assetId: string;
	    errorType?: string;
	
	    static createFrom(source: any = {}) {
	        return new AssetUninstallResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.assetType = source["assetType"];
	        this.assetId = source["assetId"];
	        this.errorType = source["errorType"];
	    }
	}
	
	export class ConfigPathValidation {
	    isConfigured: boolean;
	    metroMakerDataPathValid: boolean;
	    executablePathValid: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ConfigPathValidation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isConfigured = source["isConfigured"];
	        this.metroMakerDataPathValid = source["metroMakerDataPathValid"];
	        this.executablePathValid = source["executablePathValid"];
	    }
	}
	export class DeepLinkTarget {
	    type: string;
	    id: string;
	
	    static createFrom(source: any = {}) {
	        return new DeepLinkTarget(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.id = source["id"];
	    }
	}
	export class DeepLinkResponse {
	    status: string;
	    message: string;
	    target?: DeepLinkTarget;
	
	    static createFrom(source: any = {}) {
	        return new DeepLinkResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.target = this.convertValues(source["target"], DeepLinkTarget);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class DownloadCountsByAssetTypeResponse {
	    status: string;
	    message: string;
	    assetType: string;
	    counts: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new DownloadCountsByAssetTypeResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.assetType = source["assetType"];
	        this.counts = source["counts"];
	    }
	}
	export class Favorites {
	    authors: string[];
	    maps: string[];
	    mods: string[];
	
	    static createFrom(source: any = {}) {
	        return new Favorites(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.authors = source["authors"];
	        this.maps = source["maps"];
	        this.mods = source["mods"];
	    }
	}
	export class GalleryImageResponse {
	    status: string;
	    message: string;
	    imageUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new GalleryImageResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.imageUrl = source["imageUrl"];
	    }
	}
	export class GameRunningResponse {
	    status: string;
	    message: string;
	    running: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GameRunningResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.running = source["running"];
	    }
	}
	export class GameVersionResponse {
	    status: string;
	    message: string;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new GameVersionResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.version = source["version"];
	    }
	}
	export class GenericResponse {
	    status: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new GenericResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	    }
	}
	export class GithubTokenValidResponse {
	    status: string;
	    message: string;
	    valid: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GithubTokenValidResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.valid = source["valid"];
	    }
	}
	export class InstalledMapInfo {
	    id: string;
	    version: string;
	    config: ConfigData;
	
	    static createFrom(source: any = {}) {
	        return new InstalledMapInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.version = source["version"];
	        this.config = this.convertValues(source["config"], ConfigData);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InstalledMapsResponse {
	    status: string;
	    message: string;
	    maps: InstalledMapInfo[];
	
	    static createFrom(source: any = {}) {
	        return new InstalledMapsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.maps = this.convertValues(source["maps"], InstalledMapInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InstalledModInfo {
	    id: string;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new InstalledModInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.version = source["version"];
	    }
	}
	export class InstalledModsResponse {
	    status: string;
	    message: string;
	    mods: InstalledModInfo[];
	
	    static createFrom(source: any = {}) {
	        return new InstalledModsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.mods = this.convertValues(source["mods"], InstalledModInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class IntegrityVersionSource {
	    update_type: string;
	    repo: string;
	    tag: string;
	    asset_name?: string;
	    download_url?: string;
	
	    static createFrom(source: any = {}) {
	        return new IntegrityVersionSource(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.update_type = source["update_type"];
	        this.repo = source["repo"];
	        this.tag = source["tag"];
	        this.asset_name = source["asset_name"];
	        this.download_url = source["download_url"];
	    }
	}
	export class IntegrityVersionStatus {
	    is_complete: boolean;
	    errors: string[];
	    required_checks: Record<string, boolean>;
	    matched_files: Record<string, string>;
	    source: IntegrityVersionSource;
	    fingerprint: string;
	    checked_at: string;
	
	    static createFrom(source: any = {}) {
	        return new IntegrityVersionStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.is_complete = source["is_complete"];
	        this.errors = source["errors"];
	        this.required_checks = source["required_checks"];
	        this.matched_files = source["matched_files"];
	        this.source = this.convertValues(source["source"], IntegrityVersionSource);
	        this.fingerprint = source["fingerprint"];
	        this.checked_at = source["checked_at"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class IntegrityListing {
	    has_complete_version: boolean;
	    latest_semver_version?: string;
	    latest_semver_complete?: boolean;
	    complete_versions: string[];
	    incomplete_versions: string[];
	    versions: Record<string, IntegrityVersionStatus>;
	
	    static createFrom(source: any = {}) {
	        return new IntegrityListing(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.has_complete_version = source["has_complete_version"];
	        this.latest_semver_version = source["latest_semver_version"];
	        this.latest_semver_complete = source["latest_semver_complete"];
	        this.complete_versions = source["complete_versions"];
	        this.incomplete_versions = source["incomplete_versions"];
	        this.versions = this.convertValues(source["versions"], IntegrityVersionStatus, true);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class UpdateConfig {
	    type: string;
	    repo?: string;
	    url?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.repo = source["repo"];
	        this.url = source["url"];
	    }
	}
	export class MapManifest {
	    schema_version: number;
	    id: string;
	    name: string;
	    author: string;
	    github_id: number;
	    last_updated: number;
	    city_code: string;
	    country: string;
	    location: string;
	    population: number;
	    description: string;
	    data_source: string;
	    source_quality: string;
	    level_of_detail: string;
	    special_demand: string[];
	    // Go type: struct { Latitude float64 "json:\"latitude\""; Longitude float64 "json:\"longitude\""; Zoom float64 "json:\"zoom\""; Pitch *float64 "json:\"pitch,omitempty\""; Bearing float64 "json:\"bearing\"" }
	    initial_view_state: any;
	    tags: string[];
	    gallery: string[];
	    source: string;
	    update: UpdateConfig;
	
	    static createFrom(source: any = {}) {
	        return new MapManifest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schema_version = source["schema_version"];
	        this.id = source["id"];
	        this.name = source["name"];
	        this.author = source["author"];
	        this.github_id = source["github_id"];
	        this.last_updated = source["last_updated"];
	        this.city_code = source["city_code"];
	        this.country = source["country"];
	        this.location = source["location"];
	        this.population = source["population"];
	        this.description = source["description"];
	        this.data_source = source["data_source"];
	        this.source_quality = source["source_quality"];
	        this.level_of_detail = source["level_of_detail"];
	        this.special_demand = source["special_demand"];
	        this.initial_view_state = this.convertValues(source["initial_view_state"], Object);
	        this.tags = source["tags"];
	        this.gallery = source["gallery"];
	        this.source = source["source"];
	        this.update = this.convertValues(source["update"], UpdateConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MapsResponse {
	    status: string;
	    message: string;
	    maps: MapManifest[];
	
	    static createFrom(source: any = {}) {
	        return new MapsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.maps = this.convertValues(source["maps"], MapManifest);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ModManifest {
	    schema_version: number;
	    id: string;
	    name: string;
	    author: string;
	    github_id: number;
	    last_updated: number;
	    description: string;
	    tags: string[];
	    gallery: string[];
	    source: string;
	    update: UpdateConfig;
	
	    static createFrom(source: any = {}) {
	        return new ModManifest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schema_version = source["schema_version"];
	        this.id = source["id"];
	        this.name = source["name"];
	        this.author = source["author"];
	        this.github_id = source["github_id"];
	        this.last_updated = source["last_updated"];
	        this.description = source["description"];
	        this.tags = source["tags"];
	        this.gallery = source["gallery"];
	        this.source = source["source"];
	        this.update = this.convertValues(source["update"], UpdateConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ModsResponse {
	    status: string;
	    message: string;
	    mods: ModManifest[];
	
	    static createFrom(source: any = {}) {
	        return new ModsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.mods = this.convertValues(source["mods"], ModManifest);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PendingSubscriptionUpdate {
	    assetId: string;
	    type: string;
	    currentVersion: string;
	    latestVersion: string;
	
	    static createFrom(source: any = {}) {
	        return new PendingSubscriptionUpdate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.assetId = source["assetId"];
	        this.type = source["type"];
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	    }
	}
	export class PlatformResponse {
	    status: string;
	    message: string;
	    platform: string;
	
	    static createFrom(source: any = {}) {
	        return new PlatformResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.platform = source["platform"];
	    }
	}
	export class RegistryIntegrityReport {
	    schema_version: number;
	    generated_at: string;
	    listings: Record<string, IntegrityListing>;
	
	    static createFrom(source: any = {}) {
	        return new RegistryIntegrityReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schema_version = source["schema_version"];
	        this.generated_at = source["generated_at"];
	        this.listings = this.convertValues(source["listings"], IntegrityListing, true);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RegistryIntegrityReportResponse {
	    status: string;
	    message: string;
	    report: RegistryIntegrityReport;
	
	    static createFrom(source: any = {}) {
	        return new RegistryIntegrityReportResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.report = this.convertValues(source["report"], RegistryIntegrityReport);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ResolveConfigResponse {
	    status: string;
	    message: string;
	    config: AppConfig;
	    validation: ConfigPathValidation;
	    hasGithubToken: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ResolveConfigResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.config = this.convertValues(source["config"], AppConfig);
	        this.validation = this.convertValues(source["validation"], ConfigPathValidation);
	        this.hasGithubToken = source["hasGithubToken"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ResolveConfigResult {
	    config: AppConfig;
	    validation: ConfigPathValidation;
	    hasGithubToken: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ResolveConfigResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.config = this.convertValues(source["config"], AppConfig);
	        this.validation = this.convertValues(source["validation"], ConfigPathValidation);
	        this.hasGithubToken = source["hasGithubToken"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SandboxStatusResponse {
	    status: string;
	    message: string;
	    installed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SandboxStatusResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.installed = source["installed"];
	    }
	}
	export class SetConfigPathOptions {
	    allowAutoDetect: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SetConfigPathOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.allowAutoDetect = source["allowAutoDetect"];
	    }
	}
	export class SetConfigPathResult {
	    resolveConfigResult: ResolveConfigResult;
	    source: string;
	    autoDetectedPath?: string;
	
	    static createFrom(source: any = {}) {
	        return new SetConfigPathResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.resolveConfigResult = this.convertValues(source["resolveConfigResult"], ResolveConfigResult);
	        this.source = source["source"];
	        this.autoDetectedPath = source["autoDetectedPath"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SetConfigPathResponse {
	    status: string;
	    message: string;
	    result: SetConfigPathResult;
	
	    static createFrom(source: any = {}) {
	        return new SetConfigPathResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.result = this.convertValues(source["result"], SetConfigPathResult);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class StartupReadyResponse {
	    status: string;
	    message: string;
	    ready: boolean;
	
	    static createFrom(source: any = {}) {
	        return new StartupReadyResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.ready = source["ready"];
	    }
	}
	export class SubscriptionOperation {
	    assetId: string;
	    type: string;
	    action: string;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new SubscriptionOperation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.assetId = source["assetId"];
	        this.type = source["type"];
	        this.action = source["action"];
	        this.version = source["version"];
	    }
	}
	export class SubscriptionUpdateItem {
	    version: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new SubscriptionUpdateItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.type = source["type"];
	    }
	}
	export class SubscriptionUpdateTarget {
	    assetId: string;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new SubscriptionUpdateTarget(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.assetId = source["assetId"];
	        this.type = source["type"];
	    }
	}
	export class Subscriptions {
	    maps: Record<string, string>;
	    mods: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new Subscriptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.maps = source["maps"];
	        this.mods = source["mods"];
	    }
	}
	export class UserProfilesError {
	    profileId: string;
	    assetId: string;
	    assetType: string;
	    errorType: string;
	    downloaderErrorType?: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new UserProfilesError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profileId = source["profileId"];
	        this.assetId = source["assetId"];
	        this.assetType = source["assetType"];
	        this.errorType = source["errorType"];
	        this.downloaderErrorType = source["downloaderErrorType"];
	        this.message = source["message"];
	    }
	}
	export class SyncSubscriptionsResult {
	    status: string;
	    message: string;
	    profileId: string;
	    operations: SubscriptionOperation[];
	    errors: UserProfilesError[];
	
	    static createFrom(source: any = {}) {
	        return new SyncSubscriptionsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.profileId = source["profileId"];
	        this.operations = this.convertValues(source["operations"], SubscriptionOperation);
	        this.errors = this.convertValues(source["errors"], UserProfilesError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SystemPreferences {
	    refreshRegistryOnStartup: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SystemPreferences(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.refreshRegistryOnStartup = source["refreshRegistryOnStartup"];
	    }
	}
	export class UIPreferences {
	    theme: string;
	    defaultPerPage: number;
	    searchViewMode: string;
	
	    static createFrom(source: any = {}) {
	        return new UIPreferences(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.defaultPerPage = source["defaultPerPage"];
	        this.searchViewMode = source["searchViewMode"];
	    }
	}
	
	export class UpdateSubscriptionsRequest {
	    profileId: string;
	    assets: Record<string, SubscriptionUpdateItem>;
	    action: string;
	    forceSync: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateSubscriptionsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profileId = source["profileId"];
	        this.assets = this.convertValues(source["assets"], SubscriptionUpdateItem, true);
	        this.action = source["action"];
	        this.forceSync = source["forceSync"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UserProfile {
	    id: string;
	    uuid: string;
	    name: string;
	    uiPreferences: UIPreferences;
	    systemPreferences: SystemPreferences;
	    subscriptions: Subscriptions;
	    favorites: Favorites;
	
	    static createFrom(source: any = {}) {
	        return new UserProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.uuid = source["uuid"];
	        this.name = source["name"];
	        this.uiPreferences = this.convertValues(source["uiPreferences"], UIPreferences);
	        this.systemPreferences = this.convertValues(source["systemPreferences"], SystemPreferences);
	        this.subscriptions = this.convertValues(source["subscriptions"], Subscriptions);
	        this.favorites = this.convertValues(source["favorites"], Favorites);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UpdateSubscriptionsResult {
	    status: string;
	    message: string;
	    requestType: string;
	    hasUpdates: boolean;
	    pendingCount: number;
	    pendingUpdates: PendingSubscriptionUpdate[];
	    applied: boolean;
	    profile: UserProfile;
	    persisted: boolean;
	    operations: SubscriptionOperation[];
	    errors: UserProfilesError[];
	
	    static createFrom(source: any = {}) {
	        return new UpdateSubscriptionsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.requestType = source["requestType"];
	        this.hasUpdates = source["hasUpdates"];
	        this.pendingCount = source["pendingCount"];
	        this.pendingUpdates = this.convertValues(source["pendingUpdates"], PendingSubscriptionUpdate);
	        this.applied = source["applied"];
	        this.profile = this.convertValues(source["profile"], UserProfile);
	        this.persisted = source["persisted"];
	        this.operations = this.convertValues(source["operations"], SubscriptionOperation);
	        this.errors = this.convertValues(source["errors"], UserProfilesError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UpdateSubscriptionsToLatestRequest {
	    profileId: string;
	    apply: boolean;
	    targets?: SubscriptionUpdateTarget[];
	
	    static createFrom(source: any = {}) {
	        return new UpdateSubscriptionsToLatestRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profileId = source["profileId"];
	        this.apply = source["apply"];
	        this.targets = this.convertValues(source["targets"], SubscriptionUpdateTarget);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class UserProfileResult {
	    status: string;
	    message: string;
	    profile: UserProfile;
	    errors: UserProfilesError[];
	
	    static createFrom(source: any = {}) {
	        return new UserProfileResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.profile = this.convertValues(source["profile"], UserProfile);
	        this.errors = this.convertValues(source["errors"], UserProfilesError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class VersionInfo {
	    version: string;
	    name: string;
	    changelog: string;
	    date: string;
	    download_url: string;
	    game_version: string;
	    sha256: string;
	    downloads: number;
	    manifest?: string;
	    prerelease: boolean;
	
	    static createFrom(source: any = {}) {
	        return new VersionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.name = source["name"];
	        this.changelog = source["changelog"];
	        this.date = source["date"];
	        this.download_url = source["download_url"];
	        this.game_version = source["game_version"];
	        this.sha256 = source["sha256"];
	        this.downloads = source["downloads"];
	        this.manifest = source["manifest"];
	        this.prerelease = source["prerelease"];
	    }
	}
	export class VersionsResponse {
	    status: string;
	    message: string;
	    versions: VersionInfo[];
	
	    static createFrom(source: any = {}) {
	        return new VersionsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.versions = this.convertValues(source["versions"], VersionInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

