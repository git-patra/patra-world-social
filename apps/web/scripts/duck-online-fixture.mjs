// Integration-test bridge only. Requires existing npm tooling, no browser.
import {build} from 'esbuild';
import {mkdtempSync,readFileSync,rmSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join,dirname,resolve} from 'node:path';
import {fileURLToPath} from 'node:url';
import {spawnSync} from 'node:child_process';
const web=resolve(dirname(fileURLToPath(import.meta.url)),'..'),dir=mkdtempSync(join(tmpdir(),'duck-online-'));
try{
 const entry=join(dir,'bridge.mjs');
 await build({entryPoints:[join(web,'tests/duck-online-bridge.ts')],outfile:entry,bundle:true,platform:'node',format:'esm',loader:{'.css':'empty'},banner:{js:"import {createRequire} from 'node:module';const require=createRequire(import.meta.url);"}});
 const result=spawnSync(process.execPath,[entry],{input:readFileSync(0),encoding:'utf8',timeout:10000,maxBuffer:1024*1024});
 if(result.error)throw result.error;
 if(result.status!==0)throw Error(result.stderr||'Duck bridge failed');
 process.stdout.write(result.stdout);
}finally{rmSync(dir,{recursive:true,force:true})}
