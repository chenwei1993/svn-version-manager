export namespace model {
	
	export class AppSettings {
	    svnPath: string;
	    svnAuthor: string;
	    defaultExportDir: string;
	    logLimit: number;
	    dataDir: string;
	    configDir: string;
	    configFile: string;
	    dbPath: string;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.svnPath = source["svnPath"];
	        this.svnAuthor = source["svnAuthor"];
	        this.defaultExportDir = source["defaultExportDir"];
	        this.logLimit = source["logLimit"];
	        this.dataDir = source["dataDir"];
	        this.configDir = source["configDir"];
	        this.configFile = source["configFile"];
	        this.dbPath = source["dbPath"];
	    }
	}
	export class DBChange {
	    id: number;
	    featureId: number;
	    kind: string;
	    tableName: string;
	    columnName: string;
	    dataType: string;
	    remark: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new DBChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.featureId = source["featureId"];
	        this.kind = source["kind"];
	        this.tableName = source["tableName"];
	        this.columnName = source["columnName"];
	        this.dataType = source["dataType"];
	        this.remark = source["remark"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class ExportResult {
	    folder: string;
	    zipPath: string;
	    fileCount: number;
	    skipped: string[];
	    manifest: string;
	
	    static createFrom(source: any = {}) {
	        return new ExportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folder = source["folder"];
	        this.zipPath = source["zipPath"];
	        this.fileCount = source["fileCount"];
	        this.skipped = source["skipped"];
	        this.manifest = source["manifest"];
	    }
	}
	export class Feature {
	    id: number;
	    projectId: number;
	    name: string;
	    version: string;
	    description: string;
	    status: string;
	    createdAt: string;
	    updatedAt: string;
	    dbChangeCount: number;
	    revisionCount: number;
	    fileCount: number;
	
	    static createFrom(source: any = {}) {
	        return new Feature(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.projectId = source["projectId"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.description = source["description"];
	        this.status = source["status"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.dbChangeCount = source["dbChangeCount"];
	        this.revisionCount = source["revisionCount"];
	        this.fileCount = source["fileCount"];
	    }
	}
	export class FeatureFile {
	    id: number;
	    featureId: number;
	    revision: number;
	    path: string;
	    action: string;
	    kind: string;
	
	    static createFrom(source: any = {}) {
	        return new FeatureFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.featureId = source["featureId"];
	        this.revision = source["revision"];
	        this.path = source["path"];
	        this.action = source["action"];
	        this.kind = source["kind"];
	    }
	}
	export class FeatureRevision {
	    id: number;
	    featureId: number;
	    revision: number;
	    author: string;
	    date: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new FeatureRevision(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.featureId = source["featureId"];
	        this.revision = source["revision"];
	        this.author = source["author"];
	        this.date = source["date"];
	        this.message = source["message"];
	    }
	}
	export class FeatureDetail {
	    feature: Feature;
	    dbChanges: DBChange[];
	    revisions: FeatureRevision[];
	    files: FeatureFile[];
	
	    static createFrom(source: any = {}) {
	        return new FeatureDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.feature = this.convertValues(source["feature"], Feature);
	        this.dbChanges = this.convertValues(source["dbChanges"], DBChange);
	        this.revisions = this.convertValues(source["revisions"], FeatureRevision);
	        this.files = this.convertValues(source["files"], FeatureFile);
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
	
	
	export class Project {
	    id: number;
	    name: string;
	    path: string;
	    repoUrl: string;
	    repoRoot: string;
	    relative: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Project(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.repoUrl = source["repoUrl"];
	        this.repoRoot = source["repoRoot"];
	        this.relative = source["relative"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class SvnChangedFile {
	    path: string;
	    action: string;
	    kind: string;
	
	    static createFrom(source: any = {}) {
	        return new SvnChangedFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.action = source["action"];
	        this.kind = source["kind"];
	    }
	}
	export class SvnCommit {
	    revision: number;
	    author: string;
	    date: string;
	    message: string;
	    files: SvnChangedFile[];
	
	    static createFrom(source: any = {}) {
	        return new SvnCommit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.revision = source["revision"];
	        this.author = source["author"];
	        this.date = source["date"];
	        this.message = source["message"];
	        this.files = this.convertValues(source["files"], SvnChangedFile);
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
	export class SvnStatus {
	    available: boolean;
	    path: string;
	    version: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new SvnStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.path = source["path"];
	        this.version = source["version"];
	        this.message = source["message"];
	    }
	}

}

