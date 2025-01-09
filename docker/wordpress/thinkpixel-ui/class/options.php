<?php
/**
 * Core of ThinkPixel_*
 */

/**
 * Class for handling the Wordpress Options
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

class Options {
    const PREFIX = 'thinkpixel_';
    const OPTIONS = 'assets/opts';
    const BUFFER = 128;
    const AUTOLOAD = TRUE;

    private $options;
    private $pages;
    private $categories;

    public function get( $what = null, $opts = null ){
        switch( (string) $what ){
            case 'options':
                $out = [];

                $path = dirname( dirname( __FILE__ ) ) . DIRECTORY_SEPARATOR . self::OPTIONS . DIRECTORY_SEPARATOR;

                if( !is_dir( $path ) ) return null;
                if( ($dh = opendir( $path )) === FALSE ) return null;
                while( ($file = readdir($dh)) !== FALSE ){
                    if( $file == '.' || $file == '..' ) continue;
                    if( strtolower( substr( $file, -3 )) != 'php' ) continue;
                    $header = $this->get( 'header', $path . DIRECTORY_SEPARATOR . $file );
                    if( is_null( $header ) ) continue;
                    $out[] = [
                        'path' => get_template_directory() . DIRECTORY_SEPARATOR . self::OPTIONS . DIRECTORY_SEPARATOR . $file,
                        'slug' => self::PREFIX . strtolower( substr( $file, 0, -4 ) ),
                        'name' => $header['name'] ? : $file
                    ];
                }
                closedir( $dh );

                return $out;
                break;
            case 'header':
                $out = [];

                $state = 0;

                if( !file_exists( $opts ) ) return null;
                if( ($fh = fopen( $opts, 'r' )) === FALSE ) return null;

                while( (($line = fgets( $fh, self::BUFFER )) !== FALSE) && ($state < 2) ){
                    $line = trim( $line );
                    if( strpos( $line, '/*' ) === 0 ){ $state = 1; continue; }
                    if( strpos( $line, '*/' ) === 0 ) break;
                    if( $state < 1 ) continue;
                    list( $key, $value ) = explode( ':', $line );
                    $out[str_replace( ' ', '_', trim( strtolower( $key ) ) )] = trim( $value );
                }

                fclose ($fh);
                return $out;
                break;
            case 'value':
                if( !is_array( $opts) && is_string( $opts ) && (strpos( $opts, '.' ) !== FALSE) ){
                    list( $page, $option ) = explode( '.', $opts );
                    $opts = [ 'page' => $page, 'option' => $option, 'echo' => TRUE ];
                }

                if( !isset( $this->options[$opts['page']] ) ) $this->options[$opts['page']] = get_option( self::PREFIX . $opts['page'], [] );
                if( empty( $this->options[$opts['page']] ) || !isset( $this->options[$opts['page']][$opts['option']] ) ) return null;
                if( !$opts['echo'] ) return $this->options[$opts['page']][$opts['option']];
                echo $this->options[$opts['page']][$opts['option']];
                break;
        }
    }

    private function render( $field = null, $opts = null, $echo = false ){
        $out = '';

        switch( (string) $opts['type'] ){
            case 'page':
                if( empty( $this->pages ) ) $this->pages = get_pages();
                $out .= vsprintf( '<li>
    <label for="%s">%s</label>
    <select id="%s" name="%s">
        <option value="0">%s</option>', [
                                                $opts['id'],
                                                $opts['label'],
                                                $opts['id'],
                                                $field,
                                                $opts['noopts']
                                                ]);
                if( !empty( $this->pages ) )
                    foreach( $this->pages as $page ){
                        $out .= vsprintf( '
        <option value="%d"%s>%s</option>', [
                                                $page->ID,
                                                $page->ID == $opts['default'] ? ' selected' : '',
                                                $page->post_title
                                                ]);
                    }
                $out .= '
    </select>
</li>';
                break;
            case 'category':
                if( empty( $this->categories ) ) $this->categories = get_categories();
                $out .= vsprintf( '<li>
    <label for="%s">%s</label>
    <select id="%s" name="%s">
        <option value="0">%s</option>', [
                                                $opts['id'],
                                                $opts['label'],
                                                $opts['id'],
                                                $field,
                                                $opts['noopts']
                                                ] );
                if( !empty( $this->categories ) )
                    foreach( $this->categories as $category ){
                        $out .= vsprintf( '
        <option value="%d"%s>%s</option>', [
                                                $category->term_id,
                                                $category->term_id == $opts['default'] ? ' selected' : '',
                                                $category->cat_name
                                                ] );
                    }
                $out .= '
    </select>
</li>';
                break;
            case 'image':
                $out .= vsprintf( '<li>
    <label for="%s">%s</label>
    <input id="%s" type="text" maxlength="45" size="20" name="%s" value="%s" />
    <input id="%s_button" class="button custom-upload-button" type="button" value="Upload Image" />
</li>',[
                                                                    $opts['id'],
                                                                    $opts['label'],
                                                                    $opts['id'],
                                                                    $field,
                                                                    $opts['default'],
                                                                    $opts['id']
                                                                    ] );
                break;
            default:
                $out .= vsprintf( '<li>
                            <label for="%s">%s</label>
                            <input id="%s" type="text" maxlength="45" size="20" name="%s" value="%s" />
                    </li>', [
                                                                    $opts['id'],
                                                                    $opts['label'],
                                                                    $opts['id'],
                                                                    $field,
                                                                    $opts['default']
                                                                    ] );
                break;
        }
                    
        if( !$echo ) return $out;
        echo $out;
    }
                    
    public function page(){
        if( !isset( $_GET['page'] ) || (strpos( $_GET['page'], self::PREFIX) !== 0 ) || !preg_match( '/^[A-z_]+$/', $_GET['page'] ) ) return;
                    
        $page = substr( $_GET['page'], strlen( self::PREFIX ) );
        $file = dirname( dirname( __FILE__ ) ) . DIRECTORY_SEPARATOR . self::OPTIONS . DIRECTORY_SEPARATOR . $page . '.php';
        if( !file_exists( $file ) ) return;
        $header = $this->get( 'header', $file );
                    
        include( $file );
                    
        $saved_options = get_option( self::PREFIX . $page, [] );
        $changed_options = 0;
        $count_id = 1;
                    
        if( !empty( $options ) )
            foreach( $options as $field_name => $field_options ){
                if( !isset( $saved_options[$field_name] ) ) $saved_options[$field_name] = $field_options['default'];
                    
                $value = isset( $_POST[$field_name] ) ? $_POST[$field_name] : (isset( $saved_options[$field_name] ) ? $saved_options[$field_name] : $field_options['default']);
                if( $value != $saved_options[$field_name] ){
                    $saved_options[$field_name] = $value;
                    $changed_options ++;
                }
                    
                switch( (string)$field_options['type'] ){
                    case 'page':
                    case 'category':
                        $options[$field_name]['default'] = (int) $value;
                        break;
                    case 'image':
                        default:
                        $options[$field_name]['default'] = $value;
                        break;
                }
                $options[$field_name]['id'] = self::PREFIX . 'field_' . ($count_id++);
                                    }
                if( $changed_options )
                    update_option( self::PREFIX . $page, $saved_options, self::AUTOLOAD );
            
            
                $out = '';
            
                $out .= vsprintf( '<div class="wrap">
    <div class="icon32" id="%s"></div>
    <h2>%s</h2>
    <p>%s</p>
    <form action="" method="post">
        <ul>
            ', [
                                            $header['icon'] ? : 'icon-tools',
                                            $header['name'],
                                            $header['description']
                                            ] );
            
                if( !empty( $options ) ){
                    foreach( $options as $field_name => $field_options )
                        $out .= $this->render( $field_name, $field_options );
                }
            
                $out .= vsprintf ( '
        </ul>
        <p class="submit">
            <input type="submit" class="button-primary" value="%s" />
        </p>
    </form>
</div>', [
                    esc_attr__(/*T[*/'Save Changes'/*]*/)
                    ]);
            
        echo $out;
    }
            
    public function register( $parent_slug, $capability ){
        $options = $this->get( 'options' );
        if( !empty( $options ) )
            foreach( $options as $option )
                add_submenu_page( self::PREFIX . $parent_slug, $option['name'], $option['name'], $capability, $option['slug'], [$this, 'page'] );
    }
            
    public function __construct(){
        $this->options = [];
    }
    public function __destruct(){}
}
                                