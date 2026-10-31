package com.forum.app;

import android.os.Bundle;
import android.webkit.WebView;

import com.getcapacitor.BridgeActivity;
import com.getcapacitor.WebViewListener;

public class MainActivity extends BridgeActivity {
    @Override
    public void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        getBridge().addWebViewListener(new WebViewListener() {
            @Override
            public void onPageLoaded(WebView view) {
                view.evaluateJavascript(
                    "(function(){" +
                    "function remove(){try{document.querySelectorAll('.install-app-button,.install-app-hint').forEach(function(e){e.remove();});}catch(e){}}" +
                    "remove();" +
                    "if(window.__installObserver) return;" +
                    "if(document.body){window.__installObserver=new MutationObserver(remove);window.__installObserver.observe(document.body,{childList:true,subtree:true});}" +
                    "setTimeout(remove,500);setTimeout(remove,1500);" +
                    "})()",
                    null
                );
            }
        });
    }
}
