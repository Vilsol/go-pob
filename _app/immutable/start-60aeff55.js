var Ke=Object.defineProperty;var We=(n,e,t)=>e in n?Ke(n,e,{enumerable:!0,configurable:!0,writable:!0,value:t}):n[e]=t;var ue=(n,e,t)=>(We(n,typeof e!="symbol"?e+"":e,t),t);import{S as He,i as Fe,s as Ge,a as Ye,e as x,c as Me,b as W,g as te,t as B,d as ne,f as z,h as J,j as Xe,o as me,k as Qe,l as Ze,m as et,n as de,p as C,q as tt,r as nt,u as rt,v as H,w as be,x as F,y as G,z as Ie}from"./chunks/index-561d0633.js";import{w as ye}from"./chunks/index-fdea3aaa.js";import{a as at,s as st}from"./chunks/paths-ae0b3329.js";class Z{constructor(e,t){ue(this,"name","HttpError");ue(this,"stack");this.status=e,this.message=t!=null?t:`Error: ${e}`}toString(){return this.message}}class Te{constructor(e,t){this.status=e,this.location=t}}function it(n,e){return n==="/"||e==="ignore"?n:e==="never"?n.endsWith("/")?n.slice(0,-1):n:e==="always"&&!n.endsWith("/")?n+"/":n}function ot(n){for(const e in n)n[e]=n[e].replace(/%23/g,"#").replace(/%3[Bb]/g,";").replace(/%2[Cc]/g,",").replace(/%2[Ff]/g,"/").replace(/%3[Ff]/g,"?").replace(/%3[Aa]/g,":").replace(/%40/g,"@").replace(/%26/g,"&").replace(/%3[Dd]/g,"=").replace(/%2[Bb]/g,"+").replace(/%24/g,"$");return n}const ct=["href","pathname","search","searchParams","toString","toJSON"];function lt(n,e){const t=new URL(n);for(const o of ct){let s=t[o];Object.defineProperty(t,o,{get(){return e(),s},enumerable:!0,configurable:!0})}return t[Symbol.for("nodejs.util.inspect.custom")]=(o,s,u)=>u(n,s),ft(t),t}function ft(n){Object.defineProperty(n,"hash",{get(){throw new Error("Cannot access event.url.hash. Consider using `$page.url.hash` inside a component instead")}})}function De(n){let e=n.baseURI;if(!e){const t=n.getElementsByTagName("base");e=t.length?t[0].href:n.URL}return e}function _e(){return{x:pageXOffset,y:pageYOffset}}function Ve(n){return n.composedPath().find(t=>t instanceof Node&&t.nodeName.toUpperCase()==="A")}function Ce(n){return n instanceof SVGAElement?new URL(n.href.baseVal,document.baseURI):new URL(n.href)}function Ne(n){const e=ye(n);let t=!0;function o(){t=!0,e.update(r=>r)}function s(r){t=!1,e.set(r)}function u(r){let f;return e.subscribe(g=>{(f===void 0||t&&g!==f)&&r(f=g)})}return{notify:o,set:s,subscribe:u}}function ut(){const{set:n,subscribe:e}=ye(!1);let t;async function o(){clearTimeout(t);const s=await fetch(`${at}/_app/version.json`,{headers:{pragma:"no-cache","cache-control":"no-cache"}});if(s.ok){const{version:u}=await s.json(),r=u!=="1661745268421";return r&&(n(!0),clearTimeout(t)),r}else throw new Error(`Version check failed: ${s.status}`)}return{subscribe:e,check:o}}function dt(n){let e=5381,t=n.length;if(typeof n=="string")for(;t;)e=e*33^n.charCodeAt(--t);else for(;t;)e=e*33^n[--t];return(e>>>0).toString(36)}const ee=window.fetch;function pt(n,e){let o=`script[sveltekit\\:data-type="data"][sveltekit\\:data-url=${JSON.stringify(typeof n=="string"?n:n.url)}]`;e&&typeof e.body=="string"&&(o+=`[sveltekit\\:data-body="${dt(e.body)}"]`);const s=document.querySelector(o);if(s&&s.textContent){const{body:u,...r}=JSON.parse(s.textContent);return Promise.resolve(new Response(u,r))}return ee(n,e)}const ht=/^(\.\.\.)?(\w+)(?:=(\w+))?$/;function mt(n){const e=[],t=[];let o=!0;if(/\]\[/.test(n))throw new Error(`Invalid route ${n} \u2014 parameters must be separated`);if(qe("[",n)!==qe("]",n))throw new Error(`Invalid route ${n} \u2014 brackets are unbalanced`);return{pattern:n===""?/^\/$/:new RegExp(`^${n.split(/(?:\/|$)/).filter(_t).map((u,r,f)=>{const g=decodeURIComponent(u),h=/^\[\.\.\.(\w+)(?:=(\w+))?\]$/.exec(g);if(h)return e.push(h[1]),t.push(h[2]),"(?:/(.*))?";const w=r===f.length-1;return g&&"/"+g.split(/\[(.+?)\]/).map((R,E)=>{if(E%2){const I=ht.exec(R);if(!I)throw new Error(`Invalid param: ${R}. Params and matcher names can only have underscores and alphanumeric characters.`);const[,j,Y,M]=I;return e.push(Y),t.push(M),j?"(.*?)":"([^/]+?)"}return w&&R.includes(".")&&(o=!1),R.normalize().replace(/%5[Bb]/g,"[").replace(/%5[Dd]/g,"]").replace(/#/g,"%23").replace(/\?/g,"%3F").replace(/[.*+?^${}()|[\]\\]/g,"\\$&")}).join("")}).join("")}${o?"/?":""}$`),names:e,types:t}}function _t(n){return!/^\([^)]+\)$/.test(n)}function gt(n,e,t,o){const s={};for(let u=0;u<e.length;u+=1){const r=e[u],f=t[u],g=n[u+1]||"";if(f){const h=o[f];if(!h)throw new Error(`Missing "${f}" param matcher`);if(!h(g))return}s[r]=g}return s}function qe(n,e){let t=0;for(let o=0;o<e.length;o+=1)e[o]===n&&(t+=1);return t}function wt(n,e,t){return Object.entries(e).map(([o,[s,u,r]])=>{const{pattern:f,names:g,types:h}=mt(o),w=s<0;w&&(s=~s);const R={id:o,exec:E=>{const I=f.exec(E);if(I)return gt(I,g,h,t)},errors:[1,...r||[]].map(E=>n[E]),layouts:[0,...u||[]].map(E=>n[E]),leaf:n[s],uses_server_data:w};return R.errors.length=R.layouts.length=Math.max(R.errors.length,R.layouts.length),R})}function bt(n,e){return new Z(n,e)}function yt(n){let e,t,o;var s=n[0][0];function u(r){return{props:{data:r[1],errors:r[3]}}}return s&&(e=new s(u(n))),{c(){e&&H(e.$$.fragment),t=x()},l(r){e&&be(e.$$.fragment,r),t=x()},m(r,f){e&&F(e,r,f),W(r,t,f),o=!0},p(r,f){const g={};if(f&2&&(g.data=r[1]),f&8&&(g.errors=r[3]),s!==(s=r[0][0])){if(e){te();const h=e;B(h.$$.fragment,1,0,()=>{G(h,1)}),ne()}s?(e=new s(u(r)),H(e.$$.fragment),z(e.$$.fragment,1),F(e,t.parentNode,t)):e=null}else s&&e.$set(g)},i(r){o||(e&&z(e.$$.fragment,r),o=!0)},o(r){e&&B(e.$$.fragment,r),o=!1},d(r){r&&J(t),e&&G(e,r)}}}function vt(n){let e,t,o;var s=n[0][0];function u(r){return{props:{data:r[1],errors:r[3],$$slots:{default:[kt]},$$scope:{ctx:r}}}}return s&&(e=new s(u(n))),{c(){e&&H(e.$$.fragment),t=x()},l(r){e&&be(e.$$.fragment,r),t=x()},m(r,f){e&&F(e,r,f),W(r,t,f),o=!0},p(r,f){const g={};if(f&2&&(g.data=r[1]),f&8&&(g.errors=r[3]),f&517&&(g.$$scope={dirty:f,ctx:r}),s!==(s=r[0][0])){if(e){te();const h=e;B(h.$$.fragment,1,0,()=>{G(h,1)}),ne()}s?(e=new s(u(r)),H(e.$$.fragment),z(e.$$.fragment,1),F(e,t.parentNode,t)):e=null}else s&&e.$set(g)},i(r){o||(e&&z(e.$$.fragment,r),o=!0)},o(r){e&&B(e.$$.fragment,r),o=!1},d(r){r&&J(t),e&&G(e,r)}}}function kt(n){let e,t,o;var s=n[0][1];function u(r){return{props:{data:r[2]}}}return s&&(e=new s(u(n))),{c(){e&&H(e.$$.fragment),t=x()},l(r){e&&be(e.$$.fragment,r),t=x()},m(r,f){e&&F(e,r,f),W(r,t,f),o=!0},p(r,f){const g={};if(f&4&&(g.data=r[2]),s!==(s=r[0][1])){if(e){te();const h=e;B(h.$$.fragment,1,0,()=>{G(h,1)}),ne()}s?(e=new s(u(r)),H(e.$$.fragment),z(e.$$.fragment,1),F(e,t.parentNode,t)):e=null}else s&&e.$set(g)},i(r){o||(e&&z(e.$$.fragment,r),o=!0)},o(r){e&&B(e.$$.fragment,r),o=!1},d(r){r&&J(t),e&&G(e,r)}}}function xe(n){let e,t=n[5]&&Be(n);return{c(){e=Qe("div"),t&&t.c(),this.h()},l(o){e=Ze(o,"DIV",{id:!0,"aria-live":!0,"aria-atomic":!0,style:!0});var s=et(e);t&&t.l(s),s.forEach(J),this.h()},h(){de(e,"id","svelte-announcer"),de(e,"aria-live","assertive"),de(e,"aria-atomic","true"),C(e,"position","absolute"),C(e,"left","0"),C(e,"top","0"),C(e,"clip","rect(0 0 0 0)"),C(e,"clip-path","inset(50%)"),C(e,"overflow","hidden"),C(e,"white-space","nowrap"),C(e,"width","1px"),C(e,"height","1px")},m(o,s){W(o,e,s),t&&t.m(e,null)},p(o,s){o[5]?t?t.p(o,s):(t=Be(o),t.c(),t.m(e,null)):t&&(t.d(1),t=null)},d(o){o&&J(e),t&&t.d()}}}function Be(n){let e;return{c(){e=tt(n[6])},l(t){e=nt(t,n[6])},m(t,o){W(t,e,o)},p(t,o){o&64&&rt(e,t[6])},d(t){t&&J(e)}}}function Et(n){let e,t,o,s,u;const r=[vt,yt],f=[];function g(w,R){return w[0][1]?0:1}e=g(n),t=f[e]=r[e](n);let h=n[4]&&xe(n);return{c(){t.c(),o=Ye(),h&&h.c(),s=x()},l(w){t.l(w),o=Me(w),h&&h.l(w),s=x()},m(w,R){f[e].m(w,R),W(w,o,R),h&&h.m(w,R),W(w,s,R),u=!0},p(w,[R]){let E=e;e=g(w),e===E?f[e].p(w,R):(te(),B(f[E],1,1,()=>{f[E]=null}),ne(),t=f[e],t?t.p(w,R):(t=f[e]=r[e](w),t.c()),z(t,1),t.m(o.parentNode,o)),w[4]?h?h.p(w,R):(h=xe(w),h.c(),h.m(s.parentNode,s)):h&&(h.d(1),h=null)},i(w){u||(z(t),u=!0)},o(w){B(t),u=!1},d(w){f[e].d(w),w&&J(o),h&&h.d(w),w&&J(s)}}}function Rt(n,e,t){let{stores:o}=e,{page:s}=e,{components:u}=e,{data_0:r=null}=e,{data_1:f=null}=e,{errors:g}=e;Xe(o.page.notify);let h=!1,w=!1,R=null;return me(()=>{const E=o.page.subscribe(()=>{h&&(t(5,w=!0),t(6,R=document.title||"untitled page"))});return t(4,h=!0),E}),n.$$set=E=>{"stores"in E&&t(7,o=E.stores),"page"in E&&t(8,s=E.page),"components"in E&&t(0,u=E.components),"data_0"in E&&t(1,r=E.data_0),"data_1"in E&&t(2,f=E.data_1),"errors"in E&&t(3,g=E.errors)},n.$$.update=()=>{n.$$.dirty&384&&o.page.set(s)},[u,r,f,g,h,w,R,o,s]}class Lt extends He{constructor(e){super(),Fe(this,e,Rt,Et,Ge,{stores:7,page:8,components:0,data_0:1,data_1:2,errors:3})}}const $t=function(){const e=document.createElement("link").relList;return e&&e.supports&&e.supports("modulepreload")?"modulepreload":"preload"}(),St=function(n,e){return new URL(n,e).href},ze={},V=function(e,t,o){return!t||t.length===0?e():Promise.all(t.map(s=>{if(s=St(s,o),s in ze)return;ze[s]=!0;const u=s.endsWith(".css"),r=u?'[rel="stylesheet"]':"";if(document.querySelector(`link[href="${s}"]${r}`))return;const f=document.createElement("link");if(f.rel=u?"stylesheet":$t,u||(f.as="script",f.crossOrigin=""),f.href=s,document.head.appendChild(f),u)return new Promise((g,h)=>{f.addEventListener("load",g),f.addEventListener("error",()=>h(new Error(`Unable to preload CSS for ${s}`)))})})).then(()=>e())},Pt={},re=[()=>V(()=>import("./chunks/0-b9b3fcf1.js"),["chunks/0-b9b3fcf1.js","components/pages/_layout.svelte-01f2413c.js","assets/+layout-b76e5589.css","chunks/index-561d0633.js","chunks/global-cc4562dd.js","chunks/index-fdea3aaa.js","chunks/worker-3adb5701.js","chunks/colors-7fb82586.js","chunks/paths-ae0b3329.js","chunks/cache-4189339c.js"],import.meta.url),()=>V(()=>import("./chunks/1-fac38ab7.js"),["chunks/1-fac38ab7.js","components/pages/_error.svelte-daa650be.js","chunks/index-561d0633.js"],import.meta.url),()=>V(()=>import("./chunks/2-f1239a56.js"),["chunks/2-f1239a56.js","components/pages/_page.svelte-00944ca0.js","chunks/index-561d0633.js"],import.meta.url),()=>V(()=>import("./chunks/3-23ffd4e9.js"),["chunks/3-23ffd4e9.js","components/pages/calcs/_page.svelte-b2011c76.js","assets/+page-20077df8.css","chunks/index-561d0633.js","chunks/colors-7fb82586.js"],import.meta.url),()=>V(()=>import("./chunks/4-f64aca1f.js"),["chunks/4-f64aca1f.js","components/pages/configuration/_page.svelte-22c37296.js","chunks/index-561d0633.js","chunks/global-cc4562dd.js","chunks/index-fdea3aaa.js","chunks/worker-3adb5701.js","chunks/colors-7fb82586.js","chunks/SelectSelection-a4cdd5c4.js","assets/SelectSelection-5f47e494.css"],import.meta.url),()=>V(()=>import("./chunks/5-07c875b6.js"),["chunks/5-07c875b6.js","components/pages/import/_page.svelte-32d42f77.js","chunks/index-561d0633.js","chunks/Input-2a96b279.js","chunks/worker-3adb5701.js"],import.meta.url),()=>V(()=>import("./chunks/6-f83c106f.js"),["chunks/6-f83c106f.js","components/pages/items/_page.svelte-b7b7a46d.js","chunks/index-561d0633.js","chunks/Input-2a96b279.js"],import.meta.url),()=>V(()=>import("./chunks/7-0acea849.js"),["chunks/7-0acea849.js","components/pages/notes/_page.svelte-6bda84f7.js","chunks/index-561d0633.js"],import.meta.url),()=>V(()=>import("./chunks/8-b13a9775.js"),["chunks/8-b13a9775.js","components/pages/skills/_page.svelte-af48e130.js","assets/+page-a1ebfc6f.css","chunks/index-561d0633.js","chunks/global-cc4562dd.js","chunks/index-fdea3aaa.js","chunks/worker-3adb5701.js","chunks/Input-2a96b279.js","chunks/SelectSelection-a4cdd5c4.js","assets/SelectSelection-5f47e494.css","chunks/colors-7fb82586.js","chunks/cache-4189339c.js"],import.meta.url),()=>V(()=>import("./chunks/9-873a48cb.js"),["chunks/9-873a48cb.js","components/pages/tree/_page.svelte-3f7ee8a8.js","chunks/index-561d0633.js"],import.meta.url)],Ot={"":[2],calcs:[3],configuration:[4],import:[5],items:[6],notes:[7],skills:[8],tree:[9]};function Ut(n){n.client}const K={url:Ne({}),page:Ne({}),navigating:ye(null),updated:ut()},Je="sveltekit:scroll",q="sveltekit:index",pe=wt(re,Ot,Pt),ge=re[0],we=re[1];ge();we();let X={};try{X=JSON.parse(sessionStorage[Je])}catch{}function he(n){X[n]=_e()}function At({target:n,base:e,trailing_slash:t}){var Oe;const o=[],s={id:null,promise:null},u={before_navigate:[],after_navigate:[]};let r={branch:[],error:null,session_id:0,url:null},f=!1,g=!0,h=!1,w=1,R=null,E,I=!0,j=(Oe=history.state)==null?void 0:Oe[q];j||(j=Date.now(),history.replaceState({...history.state,[q]:j},"",location.href));const Y=X[j];Y&&(history.scrollRestoration="manual",scrollTo(Y.x,Y.y));let M=!1,ae,ve;async function ke(a,{noscroll:l=!1,replaceState:d=!1,keepfocus:i=!1,state:c={}},y){if(typeof a=="string"&&(a=new URL(a,De(document))),I)return ce({url:a,scroll:l?_e():null,keepfocus:i,redirect_chain:y,details:{state:c,replaceState:d},accepted:()=>{},blocked:()=>{}});await N(a)}async function Ee(a){const l=Pe(a);if(!l)throw new Error("Attempted to prefetch a URL that does not belong to this app");return s.promise=Se(l),s.id=l.id,s.promise}async function Re(a,l,d,i){var b,$,O;const c=Pe(a),y=ve={};let p=c&&await Se(c);if(!p&&a.origin===location.origin&&a.pathname===location.pathname&&(p=await oe({status:404,error:new Error(`Not found: ${a.pathname}`),url:a,routeId:null})),!p)return await N(a),!1;if(a=(c==null?void 0:c.url)||a,ve!==y)return!1;if(o.length=0,p.type==="redirect")if(l.length>10||l.includes(a.pathname))p=await oe({status:500,error:new Error("Redirect loop"),url:a,routeId:null});else return I?ke(new URL(p.location,a).href,{},[...l,a.pathname]):await N(new URL(p.location,location.href)),!1;else(($=(b=p.props)==null?void 0:b.page)==null?void 0:$.status)>=400&&await K.updated.check()&&await N(a);if(h=!0,d&&d.details){const{details:k}=d,S=k.replaceState?0:1;k.state[q]=j+=S,history[k.replaceState?"replaceState":"pushState"](k.state,"",a)}if(f?(r=p.state,p.props.page&&(p.props.page.url=a),E.$set(p.props)):Le(p),d){const{scroll:k,keepfocus:S}=d;if(!S){const L=document.body,U=L.getAttribute("tabindex");L.tabIndex=-1,L.focus({preventScroll:!0}),setTimeout(()=>{var _;(_=getSelection())==null||_.removeAllRanges()}),U!==null?L.setAttribute("tabindex",U):L.removeAttribute("tabindex")}if(await Ie(),g){const L=a.hash&&document.getElementById(a.hash.slice(1));k?scrollTo(k.x,k.y):L?L.scrollIntoView():scrollTo(0,0)}}else await Ie();s.promise=null,s.id=null,g=!0,p.props.page&&(ae=p.props.page);const v=p.state.branch[p.state.branch.length-1];I=((O=v==null?void 0:v.node.shared)==null?void 0:O.router)!==!1,i&&i(),h=!1}function Le(a){r=a.state;const l=document.querySelector("style[data-sveltekit]");if(l&&l.remove(),ae=a.props.page,E=new Lt({target:n,props:{...a.props,stores:K},hydrate:!0}),I){const d={from:null,to:new URL(location.href)};u.after_navigate.forEach(i=>i(d))}f=!0}async function Q({url:a,params:l,branch:d,status:i,error:c,routeId:y,validation_errors:p}){const v=d.filter(Boolean),b={type:"loaded",state:{url:a,params:l,branch:d,error:c,session_id:w},props:{components:v.map(S=>S.node.component),errors:p}};let $={},O=!1;for(let S=0;S<v.length;S+=1)$={...$,...v[S].data},(O||!r.branch.some(L=>L===v[S]))&&(b.props[`data_${S}`]=$,O=!0);if(!r.url||a.href!==r.url.href||r.error!==c||O){b.props.page={error:c,params:l,routeId:y,status:i,url:a,data:$};const S=(L,U)=>{Object.defineProperty(b.props.page,L,{get:()=>{throw new Error(`$page.${L} has been replaced by $page.url.${U}`)}})};S("origin","origin"),S("path","pathname"),S("query","searchParams")}return b}async function se({loader:a,parent:l,url:d,params:i,routeId:c,server_data_node:y}){var $,O,k,S,L;let p=null;const v={dependencies:new Set,params:new Set,parent:!1,url:!1},b=await a();if(($=b.shared)!=null&&$.load){let U=function(...m){for(const P of m){const{href:T}=new URL(P,d);v.dependencies.add(T)}};const _={};for(const m in i)Object.defineProperty(_,m,{get(){return v.params.add(m),i[m]},enumerable:!0});const A={routeId:c,params:_,data:(O=y==null?void 0:y.data)!=null?O:null,url:lt(d,()=>{v.url=!0}),async fetch(m,P){let T;typeof m=="string"?T=m:(T=m.url,P={body:m.method==="GET"||m.method==="HEAD"?void 0:await m.blob(),cache:m.cache,credentials:m.credentials,headers:m.headers,integrity:m.integrity,keepalive:m.keepalive,method:m.method,mode:m.mode,redirect:m.redirect,referrer:m.referrer,referrerPolicy:m.referrerPolicy,signal:m.signal,...P});const D=new URL(T,d).href;return U(D),f?ee(D,P):pt(T,P)},setHeaders:()=>{},depends:U,parent(){return v.parent=!0,l()}};Object.defineProperties(A,{props:{get(){throw new Error("@migration task: Replace `props` with `data` stuff https://github.com/sveltejs/kit/discussions/5774#discussioncomment-3292693")},enumerable:!1},session:{get(){throw new Error("session is no longer available. See https://github.com/sveltejs/kit/discussions/5883")},enumerable:!1},stuff:{get(){throw new Error("@migration task: Remove stuff https://github.com/sveltejs/kit/discussions/5774#discussioncomment-3292693")},enumerable:!1}}),p=(k=await b.shared.load.call(null,A))!=null?k:null}return{node:b,loader:a,server:y,shared:(S=b.shared)!=null&&S.load?{type:"data",data:p,uses:v}:null,data:(L=p!=null?p:y==null?void 0:y.data)!=null?L:null}}function $e(a,l,d){if(!d)return!1;if(d.parent&&l||a.url&&d.url)return!0;for(const i of a.params)if(d.params.has(i))return!0;for(const i of d.dependencies)if(o.some(c=>c(i)))return!0;return!1}function ie(a,l){var d,i;return(a==null?void 0:a.type)==="data"?{type:"data",data:a.data,uses:{dependencies:new Set((d=a.uses.dependencies)!=null?d:[]),params:new Set((i=a.uses.params)!=null?i:[]),parent:!!a.uses.parent,url:!!a.uses.url}}:(a==null?void 0:a.type)==="skip"&&l!=null?l:null}async function Se({id:a,url:l,params:d,route:i}){if(s.id===a&&s.promise)return s.promise;const{errors:c,layouts:y,leaf:p}=i,v=r.url&&{url:a!==r.url.pathname+r.url.search,params:Object.keys(d).filter(_=>r.params[_]!==d[_])};[...c,...y,p].forEach(_=>_==null?void 0:_().catch(()=>{}));const b=[...y,p];let $=null;const O=b.reduce((_,A,m)=>{var D;const P=r.branch[m],T=A&&((P==null?void 0:P.loader)!==A||$e(v,_.some(Boolean),(D=P.server)==null?void 0:D.uses));return _.push(T),_},[]);if(i.uses_server_data&&O.some(Boolean)){try{const _=await ee(`${l.pathname}${l.pathname.endsWith("/")?"":"/"}__data.json${l.search}`,{headers:{"x-sveltekit-invalidated":O.map(A=>A?"1":"").join(",")}});if($=await _.json(),!_.ok)throw $}catch{N(l);return}if($.type==="redirect")return $}const k=$==null?void 0:$.nodes;let S=!1;const L=b.map(async(_,A)=>{var le,Ue;if(!_)return;const m=r.branch[A],P=(le=k==null?void 0:k[A])!=null?le:null;if((!P||P.type==="skip")&&_===(m==null?void 0:m.loader)&&!$e(v,S,(Ue=m.shared)==null?void 0:Ue.uses))return m;if(S=!0,(P==null?void 0:P.type)==="error")throw P.httperror?bt(P.httperror.status,P.httperror.message):P.error;return se({loader:_,url:l,params:d,routeId:i.id,parent:async()=>{var je;const Ae={};for(let fe=0;fe<A;fe+=1)Object.assign(Ae,(je=await L[fe])==null?void 0:je.data);return Ae},server_data_node:ie(P,m==null?void 0:m.server)})});for(const _ of L)_.catch(()=>{});const U=[];for(let _=0;_<b.length;_+=1)if(b[_])try{U.push(await L[_])}catch(A){const m=A;if(m instanceof Te)return{type:"redirect",location:m.location};const P=A instanceof Z?A.status:500;for(;_--;)if(c[_]){let T,D=_;for(;!U[D];)D-=1;try{return T={node:await c[_](),loader:c[_],data:{},server:null,shared:null},await Q({url:l,params:d,branch:U.slice(0,D+1).concat(T),status:P,error:m,routeId:i.id})}catch{continue}}N(l);return}else U.push(void 0);return await Q({url:l,params:d,branch:U,status:200,error:null,routeId:i.id})}async function oe({status:a,error:l,url:d,routeId:i}){var $;const c={},y=await ge();let p=null;if(y.server){const O=await ee(`${d.pathname}${d.pathname.endsWith("/")?"":"/"}__data.json${d.search}`,{headers:{"x-sveltekit-invalidated":"1"}}),k=await O.json();if(p=($=k==null?void 0:k[0])!=null?$:null,!O.ok||(k==null?void 0:k.type)!=="data"){N(d);return}}const v=await se({loader:ge,url:d,params:c,routeId:i,parent:()=>Promise.resolve({}),server_data_node:ie(p)}),b={node:await we(),loader:we,shared:null,server:null,data:null};return await Q({url:d,params:c,branch:[v,b],status:a,error:l,routeId:i})}function Pe(a){if(a.origin!==location.origin||!a.pathname.startsWith(e))return;const l=decodeURI(a.pathname.slice(e.length)||"/");for(const d of pe){const i=d.exec(l);if(i){const c=new URL(a.origin+it(a.pathname,t)+a.search+a.hash);return{id:c.pathname+c.search,route:d,params:ot(i),url:c}}}}async function ce({url:a,scroll:l,keepfocus:d,redirect_chain:i,details:c,accepted:y,blocked:p}){const v=r.url;let b=!1;const $={from:v,to:a,cancel:()=>b=!0};if(u.before_navigate.forEach(O=>O($)),b){p();return}he(j),y(),f&&K.navigating.set({from:r.url,to:a}),await Re(a,i,{scroll:l,keepfocus:d,details:c},()=>{const O={from:v,to:a};u.after_navigate.forEach(k=>k(O)),K.navigating.set(null)})}function N(a){return location.href=a.href,new Promise(()=>{})}return{after_navigate:a=>{me(()=>(u.after_navigate.push(a),()=>{const l=u.after_navigate.indexOf(a);u.after_navigate.splice(l,1)}))},before_navigate:a=>{me(()=>(u.before_navigate.push(a),()=>{const l=u.before_navigate.indexOf(a);u.before_navigate.splice(l,1)}))},disable_scroll_handling:()=>{(h||!f)&&(g=!1)},goto:(a,l={})=>ke(a,l,[]),invalidate:a=>{var l,d;if(a===void 0){for(const i of r.branch)(l=i==null?void 0:i.server)==null||l.uses.dependencies.add(""),(d=i==null?void 0:i.shared)==null||d.uses.dependencies.add("");o.push(()=>!0)}else if(typeof a=="function")o.push(a);else{const{href:i}=new URL(a,location.href);o.push(c=>c===i)}return R||(R=Promise.resolve().then(async()=>{await Re(new URL(location.href),[]),R=null})),R},prefetch:async a=>{const l=new URL(a,De(document));await Ee(l)},prefetch_routes:async a=>{const d=(a?pe.filter(i=>a.some(c=>i.exec(c))):pe).map(i=>Promise.all([...i.layouts,i.leaf].map(c=>c==null?void 0:c())));await Promise.all(d)},_start_router:()=>{history.scrollRestoration="manual",addEventListener("beforeunload",i=>{let c=!1;const y={from:r.url,to:null,cancel:()=>c=!0};u.before_navigate.forEach(p=>p(y)),c?(i.preventDefault(),i.returnValue=""):history.scrollRestoration="auto"}),addEventListener("visibilitychange",()=>{if(document.visibilityState==="hidden"){he(j);try{sessionStorage[Je]=JSON.stringify(X)}catch{}}});const a=i=>{const c=Ve(i);c&&c.href&&c.hasAttribute("sveltekit:prefetch")&&Ee(Ce(c))};let l;const d=i=>{clearTimeout(l),l=setTimeout(()=>{var c;(c=i.target)==null||c.dispatchEvent(new CustomEvent("sveltekit:trigger_prefetch",{bubbles:!0}))},20)};addEventListener("touchstart",a),addEventListener("mousemove",d),addEventListener("sveltekit:trigger_prefetch",a),addEventListener("click",i=>{if(!I||i.button||i.which!==1||i.metaKey||i.ctrlKey||i.shiftKey||i.altKey||i.defaultPrevented)return;const c=Ve(i);if(!c||!c.href)return;const y=c instanceof SVGAElement,p=Ce(c);if(!y&&!(p.protocol==="https:"||p.protocol==="http:"))return;const v=(c.getAttribute("rel")||"").split(/\s+/);if(c.hasAttribute("download")||v.includes("external")||c.hasAttribute("sveltekit:reload")||(y?c.target.baseVal:c.target))return;const[b,$]=p.href.split("#");if($!==void 0&&b===location.href.split("#")[0]){M=!0,he(j),K.page.set({...ae,url:p}),K.page.notify();return}ce({url:p,scroll:c.hasAttribute("sveltekit:noscroll")?_e():null,keepfocus:!1,redirect_chain:[],details:{state:{},replaceState:p.href===location.href},accepted:()=>i.preventDefault(),blocked:()=>i.preventDefault()})}),addEventListener("popstate",i=>{if(i.state&&I){if(i.state[q]===j)return;ce({url:new URL(location.href),scroll:X[i.state[q]],keepfocus:!1,redirect_chain:[],details:null,accepted:()=>{j=i.state[q]},blocked:()=>{const c=j-i.state[q];history.go(c)}})}}),addEventListener("hashchange",()=>{M&&(M=!1,history.replaceState({...history.state,[q]:++j},"",location.href))});for(const i of document.querySelectorAll("link"))i.rel==="icon"&&(i.href=i.href);addEventListener("pageshow",i=>{i.persisted&&K.navigating.set(null)})},_hydrate:async({status:a,error:l,node_ids:d,params:i,routeId:c})=>{const y=new URL(location.href);let p;try{const v=(k,S)=>{const L=document.querySelector(`script[sveltekit\\:data-type="${k}"]`);return L!=null&&L.textContent?JSON.parse(L.textContent):S},b=v("server_data",[]),$=v("validation_errors",void 0),O=d.map(async(k,S)=>se({loader:re[k],url:y,params:i,routeId:c,parent:async()=>{const L={};for(let U=0;U<S;U+=1)Object.assign(L,(await O[U]).data);return L},server_data_node:ie(b[S])}));p=await Q({url:y,params:i,branch:await Promise.all(O),status:a,error:l!=null&&l.__is_http_error?new Z(l.status,l.message):l,validation_errors:$,routeId:c})}catch(v){const b=v;if(b instanceof Te){await N(new URL(v.location,location.href));return}p=await oe({status:b instanceof Z?b.status:500,error:b,url:y,routeId:c})}Le(p)}}}function Vt(n){}async function Ct({paths:n,target:e,route:t,spa:o,trailing_slash:s,hydrate:u}){const r=At({target:e,base:n.base,trailing_slash:s});Ut({client:r}),st(n),u&&await r._hydrate(u),t&&(o&&r.goto(location.href,{replaceState:!0}),r._start_router()),dispatchEvent(new CustomEvent("sveltekit:start"))}export{Vt as set_public_env,Ct as start};
