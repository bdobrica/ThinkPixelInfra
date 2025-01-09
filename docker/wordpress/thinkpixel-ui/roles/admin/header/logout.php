<?php
wp_logout();
header( 'Location:' . \ThinkPixel\Theme::HOME . '/', 303 );
exit( 1 );
