<?php
/**
 * Core of ThinkPixel_*
 */

/**
 * Plugin Class. The ThinkPixel plugin is an instance of this class.
 *
 * @category
 * @package ThinkPixel
 * @subpackage None
 * @copyright Bogdan Dobrica
 * @author Bogdan Dobrica <bdobrica @ gmail.com>
 * @version 0.1.0
 *
 */
namespace ThinkPixel;

class Plugin{
    const PluginSlug = 'thinkpixel_plugin';

    public function __construct(){
        add_action( 'after_setup_theme', [$this, 'translation'] );
        add_filter( 'locale', [$this, 'locale'] );
    }

    public function get( $key = null, $opts = null ){
        if( is_string ($key) ){
            switch( $key ){
                case 'url':
                    return plugins_url( $opts, __DIR__ );
                    break;
            }
        }
        return null;
    }

    public function out( $key = null, $opts = null ){
        $out = $this->get( $key, $opts );
        if (is_string( $out )) echo $out;
    }

    public function locale( $locale ){
        $storage = new Storage();
        $user = new User( $storage->get( 'customer' ) );
        $stored = $user->get( 'locale' );

        if (!$user->is()) return '';

        return $stored ? : $locale;
    }

    public function translation(){
        load_theme_textdomain(
            ThinkPixel_Theme::TEXTDOMAIN,
            get_template_directory() . '/lang/'
        );
    }
}
