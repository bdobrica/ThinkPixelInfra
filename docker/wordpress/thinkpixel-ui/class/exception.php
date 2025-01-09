<?php
/**
 * Core of ThinkPixel_*
 */

/**
 * Class for triggering custom exceptions.
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

class Exception extends \Exception {
    const Unknown_Error             = 0;

    private static $H = [
        self::Unknown_Error             => /*T[*/'An unknown error has occured. Check with the developers of this application.'/*]*/
    ];

    public function __construct( $code = 0, $message = null ){
        $code = (int) $code;
        if( is_null( $message ) && isset( self::$H[$code] ) )
            $message = self::$H[$code];
        $message = (string) $message;

        parent::__construct( $message, $code );
    }

    public function get( $key = null ){
        if( $key == 'code' ) return parent::getCode();
        return parent::getMessage ();
    }

    public function json(){
        echo json_encode( (object) [
            'error' => parent::getCode()
        ] );
    }

    public static function msg( $code = 0 ){
        if( isset( self::$H[$code] ) ) return self::$H[$code];
            return '';
    }
}
