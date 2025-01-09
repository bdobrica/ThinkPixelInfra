<?php
/**
 * Core of ThinkPixel_*
 */

/**
 * Storage Class
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

class Storage {
    public static $K = [
            'user',
            'player',
            'scenario',
            'locale'
            ];

    private $data;

    public function __construct () {
        $this->data = [];

        if (!session_id ())
                session_start ();

        $this->_unpack ();
    }

    public function set ($key = null, $value = null) {
        if (is_string ($key)) {
            if (in_array ($key, self::$K)) {
                if (is_null ($value))
                        unset ($this->data[$key]);
                else
                        $this->data[$key] = $value;
            }
        } else if (is_array ($key)) {
            foreach ($key as $_key => $_value) {
                if (in_array ($_key, self::$K)) {
                    if (is_null ($_value))
                        unset ($this->data[$_key]);
                    else
                        $this->data[$_key] = $_value;
                }
            }
        }
        $this->_pack ();
    }

    public function get ($key = null, $opts = null) {
        if (is_string ($key) && in_array ($key, self::$K))
            return isset ($this->data[$key]) ? $this->data[$key] : null;
    }
        
    private function _pack () {
        $current_user = wp_get_current_user ();
        if (! ($current_user instanceof WP_User) || $current_user->ID == 0) {
            if (!session_id()) return;
            $_SESSION[__CLASS__] = serialize ($this->data);
            return;
        }

        $user_meta = get_user_meta ($current_user->ID, __CLASS__, TRUE);
        if (empty ($user_meta))
            add_user_meta ($current_user->ID, __CLASS__, $this->data, TRUE);
        else
            update_user_meta ($current_user->ID, __CLASS__, $this->data);
    }

    private function _unpack () {
        $current_user = wp_get_current_user ();
        if (! ($current_user instanceof WP_User) || $current_user->ID == 0) {
            if (!session_id()) return;
            if (isset ($_SESSION[__CLASS__])) {
                $data = unserialize ($_SESSION[__CLASS__]);
                foreach ($data as $key => $value) {
                    if (in_array ($key, self::$K)) {
                        $this->data[$key] = $value;
                    }
                }
            }
            return;
        }

        $user_meta = get_user_meta ($current_user->ID, __CLASS__, TRUE);
        if (empty ($user_meta)) {
        } else {
            foreach ($user_meta as $key => $value) {
                if (in_array ($key, self::$K)) {
                    $this->data[$key] = $value;
                }
            }
        }
    }
}
