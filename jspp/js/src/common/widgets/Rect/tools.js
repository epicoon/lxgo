class SizeHolder {
    constructor() {
        this.width = undefined;
        this.height = undefined;
        this.contentWidth = null;
        this.contentHeight = null;
    }
    
    refresh(width, height) {
        if (this.width === undefined && this.height === undefined) {
            this.width = width;
            this.height = height;
            return false;
        }
        
        if (width == this.width && height == this.height)
            return false;
        
        this.width = width;
        this.height = height;
        return true;
    }
    
    refreshContent(width, height) {
        if (width == this.contentWidth && height == this.contentHeight)
            return false;

        this.contentWidth = width;
        this.contentHeight = height;
        return true;
    }
}
