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
                    "(function(){var s=document.createElement('style');" +
                    "s.innerHTML='.install-app-button,.install-app-hint{display:none !important;}';" +
                    "if(document.head)document.head.appendChild(s);})()",
                    null
                );
            }
        });
    }
}
