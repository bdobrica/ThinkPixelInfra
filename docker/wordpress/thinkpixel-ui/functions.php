<?php
spl_autoload_register( function( $class ) {
    if( strpos( $class, '\\ThinkPixel\\' ) !== 0 ) return;
    $class_file_name = strtolower(str_replace( '\\', '/', substr( $class, 10 )));
    $file = dirname( __FILE__ ) . '/class/' . $class_file_name . '.php';
    if( !file_exists( $file )) return;
    include( $file );
} );

$thinkpixel_plugin = new \ThinkPixel\Plugin();


global $thinkpixel_theme;

show_admin_bar( FALSE );

$thinkpixel_theme = new \ThinkPixel\Theme();
