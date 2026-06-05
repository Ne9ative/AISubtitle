export namespace config {
	
	export class Config {
	    mode: string;
	    model: string;
	    api_key: string;
	    source_lang: string;
	    batch_size: number;
	    context_size: number;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.model = source["model"];
	        this.api_key = source["api_key"];
	        this.source_lang = source["source_lang"];
	        this.batch_size = source["batch_size"];
	        this.context_size = source["context_size"];
	    }
	}

}

export namespace main {
	
	export class TranslateRequest {
	    videoPath: string;
	    engine: string;
	    model: string;
	    apiKey: string;
	    srcLang: string;
	    trackID: number;
	    testMode: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TranslateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.videoPath = source["videoPath"];
	        this.engine = source["engine"];
	        this.model = source["model"];
	        this.apiKey = source["apiKey"];
	        this.srcLang = source["srcLang"];
	        this.trackID = source["trackID"];
	        this.testMode = source["testMode"];
	    }
	}

}

export namespace mkv {
	
	export class Track {
	    ID: number;
	    Type: string;
	    Codec: string;
	    CodecID: string;
	    Language: string;
	    Name: string;
	    IsImageBased: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Track(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Type = source["Type"];
	        this.Codec = source["Codec"];
	        this.CodecID = source["CodecID"];
	        this.Language = source["Language"];
	        this.Name = source["Name"];
	        this.IsImageBased = source["IsImageBased"];
	    }
	}

}

