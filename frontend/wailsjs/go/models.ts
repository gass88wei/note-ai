export namespace main {
	
	export class ChatMessage {
	    id: number;
	    role: string;
	    content: string;
	    timestamp: string;
	    note_ids: string;
	    question_id: number;
	
	    static createFrom(source: any = {}) {
	        return new ChatMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.role = source["role"];
	        this.content = source["content"];
	        this.timestamp = source["timestamp"];
	        this.note_ids = source["note_ids"];
	        this.question_id = source["question_id"];
	    }
	}
	export class ChatResponse {
	    user_input: string;
	    answer: string;
	    note_ids: number[];
	
	    static createFrom(source: any = {}) {
	        return new ChatResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.user_input = source["user_input"];
	        this.answer = source["answer"];
	        this.note_ids = source["note_ids"];
	    }
	}
	export class ImportResult {
	    imported: number;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.imported = source["imported"];
	        this.message = source["message"];
	    }
	}
	export class LeannStatus {
	    available: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new LeannStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.message = source["message"];
	    }
	}
	export class Note {
	    id: number;
	    title: string;
	    content: string;
	    category: string;
	    tags: string;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new Note(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.content = source["content"];
	        this.category = source["category"];
	        this.tags = source["tags"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class TestConnectionResult {
	    success: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new TestConnectionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	    }
	}

}

