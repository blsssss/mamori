export namespace check {
	
	export class Finding {
	    code: string;
	    status: string;
	    params?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new Finding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.status = source["status"];
	        this.params = source["params"];
	    }
	}
	export class Result {
	    check: string;
	    status: string;
	    code: string;
	    params?: Record<string, string>;
	    findings: Finding[];
	    data?: any;
	    // Go type: time
	    started: any;
	    elapsedMs: number;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.check = source["check"];
	        this.status = source["status"];
	        this.code = source["code"];
	        this.params = source["params"];
	        this.findings = this.convertValues(source["findings"], Finding);
	        this.data = source["data"];
	        this.started = this.convertValues(source["started"], null);
	        this.elapsedMs = source["elapsedMs"];
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

export namespace main {
	
	export class AppInfo {
	    version: string;
	    commit: string;
	    elevated: boolean;
	    system: sysinfo.Info;
	
	    static createFrom(source: any = {}) {
	        return new AppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.commit = source["commit"];
	        this.elevated = source["elevated"];
	        this.system = this.convertValues(source["system"], sysinfo.Info);
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
	export class RunOptions {
	    policyTargets: string[];
	    eicarWaitSec: number;
	    allowElevation: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RunOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.policyTargets = source["policyTargets"];
	        this.eicarWaitSec = source["eicarWaitSec"];
	        this.allowElevation = source["allowElevation"];
	    }
	}

}

export namespace report {
	
	export class Summary {
	    version: string;
	    commit?: string;
	    system: sysinfo.Info;
	    // Go type: time
	    generated: any;
	    status: string;
	    code: string;
	    results: check.Result[];
	
	    static createFrom(source: any = {}) {
	        return new Summary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.commit = source["commit"];
	        this.system = this.convertValues(source["system"], sysinfo.Info);
	        this.generated = this.convertValues(source["generated"], null);
	        this.status = source["status"];
	        this.code = source["code"];
	        this.results = this.convertValues(source["results"], check.Result);
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

export namespace sysinfo {
	
	export class Info {
	    product: string;
	    version: string;
	    build: string;
	    arch: string;
	    host: string;
	    user: string;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.product = source["product"];
	        this.version = source["version"];
	        this.build = source["build"];
	        this.arch = source["arch"];
	        this.host = source["host"];
	        this.user = source["user"];
	    }
	}

}

