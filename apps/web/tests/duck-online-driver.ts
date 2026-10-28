import {DuckView} from '../src/features/duck/Duck';
import {OnlineDuckController} from '../src/features/duck/online';
export function viewNodes(node:any):any[]{if(!node||typeof node!=='object')return[];const children=Array.isArray(node.props?.children)?node.props.children:[node.props?.children];return[node,...children.flatMap((v:any)=>Array.isArray(v)?v.flatMap(viewNodes):viewNodes(v))]}
// Calls the actual rendered SVG pointer handler, which calls the production
// controller's shoot method. No authoritative engine coordinates are available.
export async function clickPresented(controller:OnlineDuckController,targetId:number,mode='center'){
 const frame=controller.frame();if(!frame.snapshot)throw Error('Missing delivered snapshot');
 let pending:Promise<unknown>|null=null;
 const tree=DuckView({s:frame.snapshot,aim:{x:0,y:0},now:frame.at,online:true,canShoot:true,saved:false,message:frame.message,onAim:()=>{},onShoot:a=>{pending=controller.shoot(a)},onReturn:()=>{}});
 const all=viewNodes(tree),target=all.find(n=>n.props?.['data-target']===targetId),svg=all.find(n=>n.type==='svg');
 const geometry=all.filter(n=>n.props?.['data-target']).map(n=>({id:n.props['data-target'],transform:n.props.transform}));
 if(!target)return{at:frame.at,geometry,clicked:false,calibrated:frame.calibrated,stale:frame.stale};
 const [x,y]=target.props.transform.match(/-?[\d.]+/g).map(Number);
 svg.props.onPointerDown({preventDefault(){},currentTarget:{getBoundingClientRect:()=>({left:0,top:0,width:1000,height:1000})},clientX:x,clientY:mode==='outside'?950:y});
 await pending;
 return{at:frame.at,geometry,clicked:true,calibrated:frame.calibrated,stale:frame.stale};
}
