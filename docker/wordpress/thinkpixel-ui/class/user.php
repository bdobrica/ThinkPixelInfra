<?php
/**
 * Core of ThinkPixel_*
 */

/**
 * User Class
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

class User {
    const LOCALE = '_thinkpixel_locale';

    private $ID;

    private $object;

    public function __construct( $data = null ){
        $this->ID = 0;
        $this->object = null;

        if( is_null( $data ) ){
            $user = wp_get_current_user();
            if( $user->ID ){
                $this->ID = $user->ID;
                $this->object = $user;
            }
        }
    }

    public function get( $key = null, $opts = null ){
        if( is_string( $key ) ){
            switch( $key ){
                case 'role':
                    if( $this->ID == 0 ) return 'default';
                    if( $this->object instanceof WP_User ) return 'admin';
                    break;
                case 'object':
                    return $this->object;
                    break;
                case 'locale':
                    if( $this->ID == 0 ) return '';
                    if( $this->object instanceof WP_User ) return get_user_meta( $this->object->ID, self::LOCALE, TRUE );
                    break;
            }
        }
        return $this->object->get( is_null( $key ) ? 'ID' : $key );
    }

    public function out( $key = null, $opts = null ){}

    public function set( $key = null, $value = null ){
        if( $this->object instanceof WP_User && is_string( $key ) && $key == 'locale' ){
            $locale = get_user_meta( $this->object->ID, self::LOCALE, TRUE );
            if( $locale )
                update_user_meta( $this->object->ID, self::LOCALE, $opts );
            else
                add_user_meta( $this->object->ID, self::LOCALE, $opts, TRUE );
        }
    }

    public function is( $key = null ){
        if( is_null( $key ) )
            return $this->ID > 0;
        if( is_string( $key ) && $key == 'admin' )
            return current_user_can( 'remove_users' );
        return FALSE;
    }

    public function __destruct(){}
}
