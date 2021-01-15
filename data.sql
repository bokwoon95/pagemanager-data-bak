-- handler_url
INSERT INTO pm_routes
    (url, handler_url)
VALUES
    ('/hello', '/')
ON CONFLICT (url) DO UPDATE SET handler_url = EXCLUDED.handler_url
;

-- content
INSERT INTO pm_routes
    (url, content)
VALUES
    ('/hello/', '<h1>this is hello</h1>')
ON CONFLICT (url) DO UPDATE SET template = EXCLUDED.template
;

-- template
INSERT INTO pm_routes
    (url, template)
VALUES
    ('/plainsimple', 'templates/plainsimple/index.html')
    -- ('/editor', 'templates/editor/editor.html')
    -- ,('/post-index', 'templates/plainsimple/post-index.html')
    -- ,('/cyschu', 'templates/cyschu/index.html')
ON CONFLICT (url) DO UPDATE SET content = EXCLUDED.content
;
