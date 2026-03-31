export namespace main {
	
	export class ConfigDTO {
	    raw: string;
	    address: string;
	    endpoint: string;
	    dns: string;
	
	    static createFrom(source: any = {}) {
	        return new ConfigDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.raw = source["raw"];
	        this.address = source["address"];
	        this.endpoint = source["endpoint"];
	        this.dns = source["dns"];
	    }
	}
	export class MetadataDTO {
	    folders: Record<string, Array<string>>;
	    ungrouped: string[];
	
	    static createFrom(source: any = {}) {
	        return new MetadataDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folders = source["folders"];
	        this.ungrouped = source["ungrouped"];
	    }
	}
	export class StatusInfo {
	    connected: boolean;
	    activeTunnel: string;
	    activeInterfaces: string[];
	
	    static createFrom(source: any = {}) {
	        return new StatusInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connected = source["connected"];
	        this.activeTunnel = source["activeTunnel"];
	        this.activeInterfaces = source["activeInterfaces"];
	    }
	}
	export class TunnelInfoDTO {
	    name: string;
	    filename: string;
	    folder: string;
	    address: string;
	    endpoint: string;
	
	    static createFrom(source: any = {}) {
	        return new TunnelInfoDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.filename = source["filename"];
	        this.folder = source["folder"];
	        this.address = source["address"];
	        this.endpoint = source["endpoint"];
	    }
	}

}

