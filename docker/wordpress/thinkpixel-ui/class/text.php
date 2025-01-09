<?php
/**
 * Core of ThinkPixel_*
 */

/**
 * Abstract class for storing text nodes
 *
 * @category Abstract
 * @package ThinkPixel
 * @subpackage None
 * @copyright Bogdan Dobrica
 * @author Bogdan Dobrica <bdobrica @ gmail.com>
 * @version 0.1.0
 *
 */
namespace ThinkPixel;

abstract class Text {
    const EXT               = '.html';
    const DEFAULT_FILE      = 'default';
    const ASSETS_DIR        = 'assets/txt';
    /**
     * Version string a.b.c
     * a # major release: paradigm changed
     * b # minor release: added removed methods/properties/altered tables
     * c # review: fixed bugs in already implemented methods/properties
     * @var string
     */
    public static $version = '1.0.0';

    /**
     * Human redable name of this object
     * @var string
     */
    public static $human = 'Text';

    /**
     * Db Scheme
     * @var array
     */
    public static $scheme = [];

    /**
     * The attached file, no prefix
     * @var string
     */
    public static $T;

    private $file;
    private $line;
    private $text;
    private $path;

    public function __construct( $file = null, $line = null ){
        $file = is_null( $file ) ? static::DEFAULT_FILE : $file;
        if( is_null( $line ) ){
            if( $file == static::DEFAULT_FILE )
                $line = 0;
            else {
                $dash = strrpos( $file, '-' );
                if( $dash !== FALSE ){
                    $line = substr( $file, $dash + 1 );
                    $file = substr( $file, 0, $dash );
                }
                else
                    $line = 0;
            }
        }
    
        $file = basename( $file );
    
        if( ($pos = strrpos( $file, '.' )) !== FALSE )
            $file = substr( $file, 0, $pos - 1 );
    
        $file_name = vsprintf( '%s-%d%s', [
                        $file,
                        $line,
                        static::EXT
                    ] );
    
        $dir_name = get_template_directory() . DIRECTORY_SEPARATOR . static::ASSETS_DIR . DIRECTORY_SEPARATOR . static::$T;
    
        if( !is_dir( $dir_name ) ) @mkdir( $dir_name, 0755, TRUE );
        if( !is_dir( $dir_name ) ) throw new Exception();
    
        $this->file = $file;
        $this->line = (int) $line;
        $this->path = $dir_name . DIRECTORY_SEPARATOR . $file_name;
    
        if( file_exists( $this->path ) )
            $this->text = file_get_contents( $this->path );
    }
    
    public function set( $key = null, $value = null ){
        if( is_string ($key) ){
            $slug = self::slug( $key );
            switch( $slug ){
                case 'text':
                    $this->text = $value;
                    break;
            }
        }
        $this->save();
        return TRUE;
    }
    
    public function get( $key = null, $opts = null ){
        if( is_string( $key ) ){
            $slug = self::slug( $key );
            switch( $slug ){
                case 'path':
                    return $this->path;
                case 'text':
                    return $this->text;
            }
        }
        return basename( $this->path, static::EXT );
    }
    
    public function save(){
        if( !is_dir( dirname( $this->path ) ) ) throw new Exception();
        if( !@file_put_contents ( $this->path, $this->text ) ) throw new Exception();
    }
    
    public function remove(){
        if( file_exists( $this->path ) )
        if( !@unlink( $this->path ) ) throw new Exception();
        return TRUE;
    }

    public function __destruct(){}

    public static function slug( $key ){
        return trim( preg_replace( '/[^a-z0-9]+/', '_', strtolower( trim( $key ) ) ), '_' );
    }
}
    