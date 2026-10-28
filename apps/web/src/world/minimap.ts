import type {Point} from './map';
export type Bounds=Point&{width:number;height:number};
export type CanvasRect={left:number;top:number;width:number;height:number};
export type ClosedMapZone={id:string;bounds:Bounds;entrance:Point;label?:string};

/** Map CSS client coordinates through a contain-fit image, including its letterbox.
 * Backing-store pixels, DPR and the world camera zoom do not enter this transform. */
export function mapPoint(clientPoint:Point,canvasRect:CanvasRect,contentBounds:Bounds):Point|null{
 if(![clientPoint.x,clientPoint.y,canvasRect.left,canvasRect.top,canvasRect.width,canvasRect.height,contentBounds.x,contentBounds.y,contentBounds.width,contentBounds.height].every(Number.isFinite)||canvasRect.width<=0||canvasRect.height<=0||contentBounds.width<=0||contentBounds.height<=0)return null;
 const scale=Math.min(canvasRect.width/contentBounds.width,canvasRect.height/contentBounds.height);
 const x=(clientPoint.x-canvasRect.left-(canvasRect.width-contentBounds.width*scale)/2)/scale;
 const y=(clientPoint.y-canvasRect.top-(canvasRect.height-contentBounds.height*scale)/2)/scale;
 if(x<-.000001||y<-.000001||x>contentBounds.width+.000001||y>contentBounds.height+.000001)return null;
 return{x:contentBounds.x+Math.max(0,Math.min(contentBounds.width,x)),y:contentBounds.y+Math.max(0,Math.min(contentBounds.height,y))};
}

export function mountMinimap(element:HTMLElement,{openMap}:{openMap:()=>void}){
 const contain=(event:Event)=>event.stopPropagation();
 const activate=(event:Event)=>{event.stopPropagation();openMap()};
 element.addEventListener('pointerdown',contain);element.addEventListener('pointerup',contain);element.addEventListener('click',activate);
 return()=>{element.removeEventListener('pointerdown',contain);element.removeEventListener('pointerup',contain);element.removeEventListener('click',activate)};
}
