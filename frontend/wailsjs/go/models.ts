export namespace types {
	
	export class AppConfig {
	    metroMakerDataPath?: string;
	    executablePath?: string;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.metroMakerDataPath = source["metroMakerDataPath"];
	        this.executablePath = source["executablePath"];
	    }
	}
	export class ConfigData {
	    name: string;
	    code: string;
	    description: string;
	    population: number;
	    country?: string;
	    thumbnail_bbox?: number[];
	    creator: string;
	    version: string;
	    // Go type: struct { Latitude float64 "json:\"latitude\""; Longitude float64 "json:\"longitude\""; Zoom float64 "json:\"zoom\""; Pitch *float64 "json:\"pitch\""; Bearing float64 "json:\"bearing\"" }
	    initial_view_state: any;
	
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
	        this.thumbnail_bbox = source["thumbnail_bbox"];
	        this.creator = source["creator"];
	        this.version = source["version"];
	        this.initial_view_state = this.convertValues(source["initial_view_state"], Object);
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
	export class MapExtractResponse {
	    status: string;
	    message: string;
	    config?: ConfigData;
	
	    static createFrom(source: any = {}) {
	        return new MapExtractResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
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
	    city_code: string;
	    country: string;
	    population: number;
	    description: string;
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
	        this.city_code = source["city_code"];
	        this.country = source["country"];
	        this.population = source["population"];
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
	export class ModManifest {
	    schema_version: number;
	    id: string;
	    name: string;
	    author: string;
	    github_id: number;
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
	export class ResolveConfigResult {
	    config: AppConfig;
	    validation: ConfigPathValidation;
	
	    static createFrom(source: any = {}) {
	        return new ResolveConfigResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.config = this.convertValues(source["config"], AppConfig);
	        this.validation = this.convertValues(source["validation"], ConfigPathValidation);
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
	
	export class VersionInfo {
	    version: string;
	    name: string;
	    changelog: string;
	    date: string;
	    download_url: string;
	    game_version: string;
	    sha256: string;
	    downloads: number;
	
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
	    }
	}

}

