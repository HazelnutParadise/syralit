package syiplot

import (
	"regexp"
	"strings"
)

// go-echarts renders a fixed-size, white-background chart in a centered
// container. Dropped as-is into a (possibly dark) Syralit page that produces a
// clashing white box, a horizontal scrollbar (the 900px canvas overflows a
// narrower iframe), and clipped labels. decorate rewrites the output to fit the
// iframe, blend with the page, and follow Syralit's light/dark theme.

var itemStyleRE = regexp.MustCompile(`(<div class="item"[^>]*style=")[^"]*(")`)

// decorate makes a rendered go-echarts page responsive and theme-aware:
//   - the chart container fills the iframe (no fixed px, no horizontal scroll)
//   - the background is transparent so the Syralit page shows through
//   - text recolors to match the parent page's light/dark theme
//   - the chart resizes with the iframe
func decorate(html string) string {
	html = itemStyleRE.ReplaceAllString(html, `${1}width:100%;height:100%;${2}`)
	inject := decorateCSS + decorateJS + "</body>"
	return strings.Replace(html, "</body>", inject, 1)
}

const decorateCSS = `<style>
html,body{margin:0;padding:0;height:100%;background:transparent;overflow:hidden}
.container{margin:0!important;padding:0;width:100%;height:100%;display:block!important}
.item{width:100%!important;height:100%!important;margin:0!important}
</style>`

// decorateJS runs after go-echarts' own init/setOption script (it is appended
// last), grabs the live instance via echarts.getInstanceByDom, repaints it for
// the parent theme, and keeps it sized to the iframe. allow-same-origin lets the
// sandboxed srcdoc read the parent theme; everything degrades gracefully.
const decorateJS = `<script>
(function(){
  function el(){return document.querySelector('.item')||document.querySelector('div[_echarts_instance_],div[id]');}
  function dark(){
    try{
      var d=window.parent&&window.parent.document;
      var t=d&&d.documentElement.getAttribute('data-theme');
      if(t==='dark')return true; if(t==='light')return false;
    }catch(e){}
    return !!(window.matchMedia&&window.matchMedia('(prefers-color-scheme: dark)').matches);
  }
  function paint(){
    var e=el(); if(!e||!window.echarts)return;
    var inst=echarts.getInstanceByDom(e); if(!inst)return;
    var fg=dark()?'#e6e6e6':'#222';
    var patch={backgroundColor:'transparent',textStyle:{color:fg},
      title:{textStyle:{color:fg}},legend:{textStyle:{color:fg}}};
    var opt=inst.getOption();
    // Sankey nodes are unlabeled by default — show them (in the theme color).
    if(opt&&opt.series&&opt.series.some(function(s){return s.type==='sankey';})){
      patch.series=opt.series.map(function(s){return s.type==='sankey'?{label:{show:true,color:fg}}:{};});
    }
    inst.setOption(patch,false);
    inst.resize();
  }
  function start(){
    paint();
    var e=el();
    if(e){try{new ResizeObserver(function(){var i=echarts.getInstanceByDom(e);if(i)i.resize();}).observe(document.body);}catch(_){}}
    try{
      var mq=window.matchMedia&&window.matchMedia('(prefers-color-scheme: dark)');
      if(mq&&mq.addEventListener)mq.addEventListener('change',paint);
    }catch(_){}
  }
  if(document.readyState==='loading')window.addEventListener('DOMContentLoaded',start);else start();
  window.addEventListener('resize',paint);
})();
</script>`
