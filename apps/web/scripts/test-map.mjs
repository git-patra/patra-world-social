import {build} from 'esbuild';
import {spawnSync} from 'node:child_process';
import {mkdtempSync,rmSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
const dir=mkdtempSync(join(tmpdir(),'patra-map-'));
try {await build({entryPoints:['tests/audio.test.ts','tests/settings.test.ts','tests/guest.test.ts','tests/map.test.ts','tests/app-shell.test.ts','tests/penalty-duel.test.ts','tests/race.test.ts','tests/duck.test.ts','tests/leaderboard.test.ts','tests/voice.test.ts'],outdir:dir,banner:{js:"import {createRequire} from 'node:module'; const require=createRequire(import.meta.url);"},bundle:true,platform:'node',format:'esm',outExtension:{'.js':'.mjs'},define:{'import.meta.env.DEV':'false'}});const r=spawnSync(process.execPath,['--test',join(dir,'audio.test.mjs'),join(dir,'settings.test.mjs'),join(dir,'guest.test.mjs'),join(dir,'map.test.mjs'),join(dir,'app-shell.test.mjs'),join(dir,'penalty-duel.test.mjs'),join(dir,'race.test.mjs'),join(dir,'duck.test.mjs'),join(dir,'leaderboard.test.mjs'),join(dir,'voice.test.mjs')],{stdio:'inherit'});process.exitCode=r.status??1;}finally{rmSync(dir,{recursive:true,force:true});}
