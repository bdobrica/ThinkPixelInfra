<?php
/**
 * Core of ThinkPixel_*
 */

/**
 * Abstract class for defining connections between objects and database tables.
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

abstract class Model {
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
    public static $human = 'Model';

    /**
     * Db Scheme
     * @var array
     */
    public static $scheme = array();

    /**
     * Model Requirements
     * @var array
     */
    public static $requires = array();

    /**
     * Cryptography in modulo $Crypto_ZP, with generator $Crypto_GN
     * @var integer
     */
    protected static $Crypto_ZP = 3195487;
    protected static $Crypto_GN = 297421;
    /**
     * The attached database table, no prefix
     * @var string
     */
    public static $T;
    /**
     * The attached database table structure. The ID column is added by default.
     * @var array
     */
    protected static $K = array();
    /**
     * The attached database meta table structure. No ID column is added by default.
     * The meta table is self::$T suffixed with '_meta'. Only if !empty( $M_K) then
     * the class contains a meta table. Meta keys can be used together with normal keys.
     * Meta keys have to indexes: oid( object id) and gid( group id)
     * Meta keys are only represented in lowercase!
     * @var array
     */
    protected static $M_K = array();
    /**
     * List of unique identifiers. If a database object matches them, then it will be updated.
     * @var array
     */
    protected static $U = array();
    /**
     * Additional table to handle 1:many links between this object and other objects
     * similar to groups.
     * @var array( list of ThinkPixel_Model descendants to be linked to)
     */
    protected static $L = array();
    /**
     * Additional table to handle instances of this object, relative to object types listed in
     * this array.
     * @var array( list of ThinkPixel_Model descendants that provide a unique pair {instance_key, instance_value})
     */
    protected static $I = array();
    /**
     * Pair of( actions, form elements).
     * Form elements are defined as name[:type] => label, where
     * 	name is a string containing the database key,
     *	type is a string containing the type prefixed by :
        *	label is a string containing the displayed label
        *	Common types are: text( default ), basket, checkbox, radio, select, button, submit, close, textarea, email, password, hidden, label, tos, date, seller, buyer, product, spread, matrix, file
        * @see ThinkPixel_Form::_render()
        * @var array
        */
    public static $F = array(
        'new' => array(),
        'edit' => array(),
        'view' => array(),
        'safe' => array(),
        'excerpt' => array(),
        'group' => array()
     );
    /**
     * List of column declarations for the table structure. The ID column is not added by default.
     * @var array
     */
    protected static $Q;
    /**
     * List of keys to group by this objects. Groups are uid based. Different uid's have different groups.
     */
    protected static $G;
    /**
     * The object's database ID
     * @var int
     */
    protected $ID;
    /**
     * The object's data. Represented as a hash table.
     * @var array
     */
    protected $data;
    protected $group;
    private $debug;

	public function __construct( $data = null ){
		global $wpdb;

		if( is_null( $data ) ){

        } else if( is_numeric($data) ){
			$row = $wpdb->get_row( $wpdb->prepare( 'select * from `' . $wpdb->prefix . static::$T . '` where id=%d;', (int) $data ), ARRAY_A );
			if( !empty($row) ){
				$this->ID = (int) $data;
				$this->data = $row;
            }
			else
				throw new Exception();
        } else if( is_string( $data ) ){
			if( strpos( $data, get_class( $this ) . '-' ) === 0 ){
				if( ($pos = strpos( $data, ':' ) ) !== FALSE ){
					$name = substr( $data, $pos + 1 );
					$data = substr( $data, 0, $pos );
                }
				if( ($pos = strpos( $data, '-' ) ) !== FALSE ){
					$data = (int) substr( $data, $pos + 1 );

					if( $data ){
						$row = $wpdb->get_row( $wpdb->prepare( 'select * from `' . $wpdb->prefix . static::$T . '` where id=%d;', $data ), ARRAY_A );
						if( !empty($row) ){
							$this->ID = $data;
							$this->data = $row;
                        }
						else
							throw new Exception();
                    }
                }
            }
			if( in_array( 'series', static::$K ) && in_array( 'number', static::$K ) && preg_match( '/^[A-z]+[0-9]+$/', $data ) ){
				$row = $wpdb->get_row( $wpdb->prepare( 'select * from `' . $wpdb->prefix . static::$T . '` where series=%s and number=%d;', array( 
					self::parse( 'series', $data ),
					self::parse( 'number', $data)
					) ), ARRAY_A );
				if( !empty( $row ) ){
					$this->ID = (int) $row['id'];
					$this->data = $row;
                }
				else
					throw new Exception();
            }
        } else if( is_array($data) ){
			foreach( static::$K as $key)
				if( isset($data[$key]))
					$this->data[$key] = $data[$key];

			if( isset($data['id']) ){
				$this->ID =( int) $data['id'];
				/**
				 * Correct behaviour, but needs to be closely analyzed as it breaks the site.
				 */
				if( $this->ID ){
					$sql = $wpdb->prepare( 'select * from `' . $wpdb->prefix . static::$T . '` where id=%d;', $this->ID );
					$row = $wpdb->get_row( $sql, ARRAY_A );
					if( !empty($row) ){
						$this->data = $row;
						return;
                    }
					throw new Exception();
                }
				/** */
            }

			/**
			 * Also the correct behaviour, but needs analysis.
			 */
			if( !empty(static::$U) ){
				$pieces = array();
				$values = array();
				$can_identify = TRUE;

				foreach( static::$U as $key ){
					$pieces[] = '`' . $key . '`=%s';
					$values[] = $this->data[$key];
					if( !isset( $this->data[$key] ) ) $can_identify = FALSE;
                }

				if( $can_identify ){
					$sql = $wpdb->prepare( 'select * from `' . $wpdb->prefix . static::$T . '` where ' . implode( ' and ', $pieces) . ';', $values );
					$row = $wpdb->get_row( $sql, ARRAY_A );
					if( !empty( $row ) ){
						$this->ID = (int) $row['id'];
						$this->data = $row;
						return;
                    }
					throw new Exception();
                }
            }
			/** */
        }
    }

	public static function slug( $key ){
		return trim( preg_replace('/[^a-z]+/', '_', strtolower( trim( $key ) ) ), '_' );
    }

	public static function _unserialize( $data ){
		if( !is_string( $data ) ) return $data;
		if( preg_match( '/^a:\d+:{.*?}$/', $data ) ){
			$out = unserialize( $data );
			return is_array( $out) ? $out : $data;
        }
		if( preg_match( '/^o:\d+:"[a-z0-9_]+":\d+:{.*?}$/', $data ) ){
			$out = unserialize( $data );
			return is_object( $out) ? $out : $data;
        }
		return $data;
    }

	private function _meta_get( $key = null, $opts = null ){
		global $wpdb;

		$slug = static::slug( $key );
		if( empty( static::$M_K ) || (!in_array( $slug, static::$M_K ) ) )
			return FALSE;
		if( isset($this->data[$slug]) )
			return $this->data[$slug];
		
		$sql = $wpdb->prepare( 'select meta_value from `' . $wpdb->prefix . static::$T . '_meta` where oid=%d and meta_key=%s;', array( 
				$this->ID,
				$slug
				) );
		$values = $wpdb->get_col( $sql );

		if( empty( $values))
			return null;

		if( count( $values) == 1)
			return $this->data[$slug] = self::_unserialize( $values[0] );

		$this->data[$slug] = array();
		foreach( $values as $value)
			$this->data[$slug][] = self::_unserialize( $value );

		return $this->data[$slug];
    }

	public function get( $key = null, $opts = null ){
		if( is_null( $key ) ) return $this->ID;
		$slug = static::slug( $key );
		if( $slug == 'self' )
			return get_class( $this ) . '-' . $this->ID;
		if( $slug == 'keys' )
			return static::$K;
		if( $slug == 'class' )
			return get_class( $this );
		if( isset( $this->data[$slug] ) )
			return $this->data[$slug];
		#if( in_array( $slug, static::$K))
		#	return( string) $this->data[$slug];
		$value = $this->_meta_get( $key, $opts );
		return $value !== FALSE ? $value : $this->ID;

		return $this->ID;
    }

	public function out( $key = null, $opts = null, $callback = null ){
		$content = $this->get( $key, $opts );
		if( !is_null( $callback) && is_callable( $callback))
			$content = call_user_func( $callback, $content );
		echo $content;
    }

	public function changes(){
		$changes = array();

		$keys = static::$F['edit'];
		if( empty( $keys ) ) $keys = static::$F['new'];
		if( empty( $keys ) ) return json_encode( $changes );

		foreach( $keys as $key_type => $key_label ){
			if( strpos( $key_type, ':' ) !== FALSE ){
				list( $key, $type ) = explode( ':', $key_type );
            }
			else {
				$key = $key_type;
				$type = '';
            }
        }

		return json_encode( $changes );
    }

	private function _meta_set( $key = null, $value = null, $single = TRUE ){
		global $wpdb;
		if( is_null( $key ) ) return FALSE;

		$slug = static::slug( $key );
		if( empty( static::$M_K ) ||( !in_array( $slug, static::$M_K ) ) )
			return FALSE;

		if( !$this->ID ){
			if( $single )
				$this->data[$slug] = $value;
			else
				$this->data[$slug][] = $value;
			return TRUE;
        }

		if( $single ){
			$sql = $wpdb->prepare( 'select id from `' . $wpdb->prefix . static::$T . '_meta` where oid=%d and meta_key=%s;', array( 
					$this->ID,
					$slug
					) );
			$ids = $wpdb->get_col( $sql );
			if( empty( $ids ) ){
				$sql = $wpdb->prepare( 'insert into `' . $wpdb->prefix . static::$T . '_meta`( oid, meta_key, meta_value) values( %d, %s, %s );', array( 
						$this->ID,
						$slug,
						(is_array( $value ) || is_object( $value ) ) ? serialize( $value ) : $value
						) );

				if( !define( 'ThinkPixel_DEBUG' ) )
					$wpdb->query( $sql );
				else
					ThinkPixel_Model::debug( $sql );
				
				return TRUE;
            }
			if( count( $ids ) == 1 ){
				$sql = $wpdb->prepare( 'update `' . $wpdb->prefix . static::$T . '_meta` set meta_value=%s where oid=%d and meta_key=%s;', array( 
						(is_array( $value) || is_object( $value)) ? serialize( $value) : $value,
						$this->ID,
						$slug
						) );

				if( !defined( 'ThinkPixel_DEBUG' ) )
					$wpdb->query( $sql );
				else
                    ThinkPixel_Model::debug( $sql );

				return TRUE;
            }
			reset( $ids );

			$count = 0;
			foreach( $ids as $id ){
				if( $count < 1 ){
					$sql = $wpdb->prepare( 'update `' . $wpdb->prefix . static::$T . '_meta` set meta_value=%s where id=%d;', array( 
							(is_array( $value ) || is_object( $value )) ? serialize( $value ) : $value,
							$id
							) );
					if( !defined( 'ThinkPixel_DEBUG' ) )
						$wpdb->query( $sql );
					else
                        ThinkPixel_Model::debug( $sql );
                }
				else {
					$sql = $wpdb->prepare( 'delete from `' . $wpdb->prefix . static::$T . '_meta` where id=%d;', $id );
					if( !defined( 'ThinkPixel_DEBUG' ) )
						$wpdb->query( $sql );
					else
                        ThinkPixel_Model::debug( $sql );
                }
				$count ++;
            }
        } else {
			$sql = $wpdb->prepare( 'insert into `' . $wpdb->prefix . static::$T . '_meta`( oid, meta_key, meta_value) values( %d, %s, %s );', array( 
					$this->ID,
					$slug,
					(is_array( $value ) || is_object( $value ) ) ? serialize( $value ) : $value
					) );
			if( !defined( 'ThinkPixel_DEBUG' ) )
				$wpdb->query( $sql );
			else
                ThinkPixel_Model::debug( $sql );
			return TRUE;
        }
    }

	public function set( $key = null, $value = null ){
		global $wpdb;

		if( is_array( $key ) ){
			if( !empty( $key ) ){
				$keys = $key;
				$update = array();
				$values = array();
			
				foreach( $keys as $key => $value ){
					$slug = static::slug( $key );
					if( in_array($slug, static::$M_K ) ){
						$this->_meta_set( $key, $value );
						continue;
                    }
					if( !in_array($slug, static::$K ) )
						continue;
					$update[] = '`' . $slug . '`=%s';
					$values[] = $value;
					$this->data[$slug] = $value;
                }

				$values[] = $this->ID;
				$sql = $wpdb->prepare( 
					'update `' . $wpdb->prefix . static::$T . '` set ' . implode( ',', $update ) . ' where id=%d;',
					$values
					);
				if( !defined( 'ThinkPixel_DEBUG' ) )
					$wpdb->query( $sql );
				else
                    ThinkPixel_Model::debug( $sql );
            }
        } else {
			$slug = static::slug( $key );

			if( $slug == 'debug' ){
				$this->debug = is_null( $value ) ? TRUE : $value;
				return TRUE;
            }

			if( in_array( $slug, static::$M_K ) ){
				$this->_meta_set( $key, $value );
				return;
            }
			if( !in_array($slug, static::$K ) ) throw new Exception();
			$this->data[$slug] = $value;

			if( $this->ID ){
				if( is_object( $value ) &&( $value instanceof ThinkPixel_Model ) )
					$value = $value->get( );
				$sql = $wpdb->prepare( 'update `' . $wpdb->prefix . static::$T . '` set `' . $slug . '`=%s where id=%d;', $value, $this->ID );
				if( !defined( 'ThinkPixel_DEBUG'))
					$wpdb->query( $sql );
				else
                    ThinkPixel_Model::debug( $sql );
            }
        }
    }

	public function field( $key, $context = 'edit' ){
		$out = array( 
			'info' => '',
			'label' => ''
			);

		if( empty( static::$F[$context] ) ) return $out;

		foreach( static::$F[$context] as $info => $label ){
			if( strpos( $info, $key ) === 0 ){
				$seps = array();
				if( ($sep = strpos( $info, '?' ) ) !== FALSE ) $seps[] = $sep;
				if( ($sep = strpos( $info, ':' ) ) !== FALSE ) $seps[] = $sep;
				if( ($sep = strpos( $info, ';' ) ) !== FALSE ) $seps[] = $sep;
				$sep = !empty( $seps) ? min( $seps) : 0;
				if( ($sep == 0) || (($sep > 0) && (substr( $info, 0, $sep ) == $key)) ){
					$out['info'] = $info;
					$out['label'] = $label;
					break;
                }
            }
        }
		return $out;
    }

	public function json( $data = FALSE ){
		if( is_string( $data ) && in_array( $data, array_keys( static::$F ) ) && !empty( static::$F[$data] ) ){
			$out = array( 'id' => $this->ID, 'self' => get_class( $this ) . '-' . $this->ID );
			foreach( static::$F[$data] as $key => $label )
				$out[$key] = $this->data[$key];
			return json_encode( (object) $out );
        }

		$out = array( 
			'type' => 'object',
			'class' => get_class( $this ),
			'id' => $this->ID
            );

		if( $data )
			$out['data'] = $this->data;

		return json_encode( $out );
    }

	/**
	 * Save handles the database INSERT / UPDATE for current object.
	 */
	public function save(){
		global $wpdb;

		if( $this->ID ) throw new Exception();

		if( in_array( 'uid', static::$K ) && !$this->data['uid'] ){
			try {
				$user = new User( FALSE );
				$this->data['uid'] = $user->get();
				if( in_array( 'oid', static::$K) && !$this->data['oid'] ){
					$this->data['oid'] = current( $user->get( 'offices') );
                }
            }
			catch( ThinkPixel_Exception $wp_crm_exception ){}
        }

		if( !empty( static::$U ) ){
			$pieces = array();
			$values = array();
			foreach( static::$U as $key ){
				$pieces[] = '`' . $key . '`=%s';
				$values[] = $this->data[$key];
            }

			$sql = $wpdb->prepare( 'select id from `' . $wpdb->prefix . static::$T . '` where ' . implode( ' and ', $pieces) . ';', $values );
			$id = $wpdb->get_var( $sql );

			if( !is_null( $id ) ){
				$this->ID = (int) $id;
				$pieces = array();
				$values = array();
				foreach( static::$K as $key ){
					$pieces[] = '`' . $key . '`=%s';
					$values[] = $this->data[$key];
                }
				$values[] = $this->ID;
				$sql = $wpdb->prepare( 'update `' . $wpdb->prefix . static::$T . '` set ' . implode( ',', $pieces) . ' where id=%d;', $values );
				if( !defined( 'ThinkPixel_DEBUG') ){
					if( $wpdb->query( $sql) !== FALSE) return;
                }
				else
					ThinkPixel_Model::debug( $sql );
            }
        }

		if( !empty( static::$K ) ){
			$columns = array();
			$formats = array();
			$values = array();
			foreach( static::$K as $key ){
				if( !isset( $this->data[$key] ) ) continue;
				$columns[] = $key;
				$formats[] = '%s';
				$values[] = is_array( $this->data[$key] ) ? serialize( $this->data[$key] ) :( string) $this->data[$key];
            }
			$sql = $wpdb->prepare( 'insert into `' . $wpdb->prefix . static::$T . '`( `' . implode('`,`', $columns) . '`) values( ' . implode( ',', $formats) . ' );', $values );
			if( !defined( 'ThinkPixel_DEBUG' ) ){
				$wpdb->query( $sql );
				if( !($this->ID = $wpdb->insert_id) ) throw new Exception();
            } else {
				$this->ID = rand();
				ThinkPixel_Model::debug( $sql );
            }
        }

		/**
		 * Meta Key
		 */
		if( $this->ID && !empty( static::$M_K ) ){
			$meta_set = array();
			foreach( static::$M_K as $meta_key )
				if( isset( $this->data[$meta_key] ) )
					$this->_meta_set( $meta_key, $this->data[$meta_key] );
        }
    }

	protected function _link_set( $class, $object, $link ){
		return FALSE;
    }

	protected function _link_get( $class, $object ){
		return FALSE;
    }

	public function link( $object = null, $link = null ){
		if( !is_object( $object ) ) return FALSE;
		$class = get_class( $object );

		if( is_null( $link ) ){
			return $this->_link_get( $class, $object );
        } else {
			return $this->_link_set( $class, $object, $link );
        }
		return FALSE;
	}

	public static function parse( $key = null, $from = null ){
		switch( $key ){
			case 'series':
				return trim( preg_replace( '/[^A-Z]+/','',strtoupper( $from ) ) );
				break;
			case 'number':
				return intval( preg_replace( '/[^0-9]+/','',$from ) );
				break;
			case 'spell number':
				$words = array( 
					1 => array( 'unu', 'doi', 'trei', 'patru', 'cinci', 'sase', 'sapte', 'opt', 'noua' ),
					10 => array( 'zece', 'douazeci', 'treizeci', 'patruzeci', 'cincizeci', 'saizeci', 'saptezeci', 'optzeci', 'nouazeci' ),
					100 => array( 'o suta', 'doua sute', 'trei sute', 'patru sute', 'cinci sute', 'sase sute', 'sapte sute', 'opt sute', 'noua sute' ),
					1000 => array( 'o mie', 'doua mii', 'trei mii', 'patru mii', 'cinci mii', 'sase mii', 'sapte mii', 'opt mii', 'noua mii' )
					);

				$integer = intval( $number );
				$decimal = intval( 100 *( $number - $integer) );

				$out = '';

				$value = $integer % 100;

				if( $value ){
					if( $value < 10 ) $out = $words[1][$value - 1] . ' ' . $out;
					else
					if( $value == 10 ) $out = $words[10][0] . $out;
					else
					if( $value < 20 ) $out = $words[1][$value%10 - 1] . 'sprezece ' . $out;
					else {
						if( $value % 10 )
							$out = $words[10][intval($value/10) - 1] . ' si ' . $words[1][$value%10 - 1] . ' ' . $out;
						else
							$out = $words[10][intval($value/10) - 1] . ' ' . $out;
                    }
                }

				if( $integer ) $out .= ( $value > 0 || $value < 20 ) ? 'lei' : 'de lei';

				$integer = intval( $integer / 100 );
				$value = $integer % 10;

				if( $value) $out = $words[100][$value - 1] . ' ' . $out;
				$integer = intval( $integer / 10 );
				$value = $integer % 10;

				if( $value ) $out = $words[1000][$value - 1] . ' ' . $out;

				if( $decimal ){
					if( $decimal < 10 ) $out .= ' si ' . $words[1][$decimal - 1];
					else
					if( $decimal == 10 ) $out .= ' si ' . $words[10][0];
					else
					if( $decimal < 20 ) $out .= ' si ' . $words[1][$decimal%10 - 1] . 'sprezece';
					else {
						if( $decimal % 10 )
							$out .= ' si ' . $words[10][intval($decimal/10) - 1] . ' si ' . $words[1][$decimal%10 - 1];
						else
							$out .= ' si ' . $words[10][intval($decimal/10) - 1];
                    }
					$out .=( $decimal < 20) ? ' bani' : ' de bani';
                }

				return str_replace( 'unusprezece', 'unsprezece', $out );
				break;
			default:
				return null;
        }
    }

	public static function install( $uninstall = FALSE ){
		global $wpdb;

		if( empty( static::$T ) ) return;
		if( empty( static::$Q ) ) return;

		$sql = $uninstall ?
			'drop table `' . $wpdb->prefix . static::$T . '`;' :
			'create table `' . $wpdb->prefix . static::$T . '`( ' . implode( ',', static::$Q ) . ') engine=MyISAM default charset=utf8;';

		if( !($uninstall xor( $wpdb->get_var( 'show tables like \'' . $wpdb->prefix . static::$T . '\';' ) == ( $wpdb->prefix . static::$T ) ) ) ){
			if( !defined( 'ThinkPixel_DEBUG' ) )
				$wpdb->query( $sql );
			else
                ThinkPixel_Model::debug( $sql );
        }

		/**
		 * Create META table if needed.
		 */
		if( !empty( static::$M_K ) ){
			$sql = $uninstall ?
				'drop table `' . $wpdb->prefix . static::$T . '_meta`;' :
				'create table `' . $wpdb->prefix . static::$T . '_meta`( 
					`id` int(11) NOT NULL PRIMARY KEY AUTO_INCREMENT,
					`oid` int(11) NOT NULL DEFAULT 0,
					`gid` int(11) NOT NULL DEFAULT 0,
					`meta_key` varchar(64) NOT NULL DEFAULT \'\',
					`meta_value` text NOT NULL,
					KEY `oid`( `oid` ),
					KEY `gid`( `gid` ),
					KEY `meta_key`( `meta_key`)
					) engine MyISAM default charset=utf8;';
			if( !($uninstall xor( $wpdb->get_var( 'show tables like \'' . $wpdb->prefix . static::$T . '_meta\';' ) == ( $wpdb->prefix . static::$T . '_meta' ) ) ) ){
				if( !defined( 'ThinkPixel_DEBUG' ) )
					$wpdb->query( $sql );
				else
					ThinkPixel_Model::debug( $sql );
            }
        }

		/**
		 * Create the link table.
		 */
		if( !empty( static::$L ) ){
			$sql = $uninstall ?
				'drop table `' . $wpdb->prefix . static::$T . '_link`;' :
				'create table `' . $wpdb->prefix . static::$T . '_link`( 
					`id` int(11) NOT NULL PRIMARY KEY AUTO_INCREMENT,
					`lid` int(11) NOT NULL DEFAULT 0,
					`object` varchar(64) NOT NULL DEFAULT \'\',
					`oid` int(11) NOT NULL DEFAULT 0,
					KEY `lid`( `oid` ),
					KEY `object`( `object`)
					) engine MyISAM default charset=utf8;';
			if( !($uninstall xor( $wpdb->get_var( 'show tables like \'' . $wpdb->prefix . static::$T . '_link\';' ) == ( $wpdb->prefix . static::$T . '_link' ) ) ) ){
				if( !defined( 'ThinkPixel_DEBUG'))
					$wpdb->query( $sql );
				else
					ThinkPixel_Model::debug( $sql );
            }
        }

		/**
		 * Create the instance table.
		 */
		if( !empty( static::$I ) ){
			$object_rows = array();
			foreach( static::$I as $class_name )
				$object_rows[] = sprintf( '`%s_id` int(11) NOT NULL DEFAULT 0,', strtolower( $class_name ) );

			$sql = $uninstall ?
				'drop table `' . $wpdb->prefix . static::$T . '_instance`;' :
				'create table `' . $wpdb->prefix . static::$T . '_instance`( 
					`id` int(11) NOT NULL PRIMARY KEY AUTO_INCREMENT,
					`rid` int(11) NOT NULL DEFAULT 0,
					' . implode( "\n", $object_rows ) . '
					`value` text NOT NULL,
					KEY `rid`( `rid`)
					) engine MyISAM default charset=utf8;';
			if( !($uninstall xor( $wpdb->get_var( 'show tables like \'' . $wpdb->prefix . static::$T . '_instance\';' ) != ( $wpdb->prefix . static::$T . '_instance' ) ) ) ){
				if( !defined( 'ThinkPixel_DEBUG' ) )
					$wpdb->query( $sql );
				else
                    ThinkPixel_Model::debug( $sql );
            }
        }
    }

	public static function upgrade(){
		global $wpdb;
		
		if( empty( static::$T ) ) return;
		if( empty( static::$Q ) ) return;

		$sql = 'show create table `' . $wpdb->prefix . static::$T . '`;';

		list( , $structure ) = $wpdb->get_row( $sql, ARRAY_N );
		$structure = array_slice( explode( "\n", $structure ), 1, -1 );

		/**
		 * TODO: when the table structure changes, update the table
		 */

		$columns_add = array();
		$previous_column = '';
		foreach( static::$Q as $new_column_def ){
			if( strpos( $new_column_def, '`' ) !== 0 ) continue;

			list( $new_column, ) = explode( ' ', $new_column_def );
			$is_new_column = TRUE;
			foreach( $structure as $old_column_def ){
				if( strpos( $old_column_def, $new_column) === FALSE ) continue;
				$is_new_column = FALSE;
				break;
            }

			if( $is_new_column )
				$columns_add[] = 'add column ' . $new_column_def .( $previous_column ? ( ' after ' . $previous_column ) : ' first' );

			$previous_column = $new_column;
        }

		if( !empty( $columns_add) ){
			$sql = 'alter table `' . $wpdb->prefix . static::$T . '` ' . implode( ', ', $columns_add ) . ';';
			if( !defined( 'ThinkPixel_DEBUG' ) )
				$wpdb->query( $sql );
			else
				ThinkPixel_Model::debug( $sql );
        }

		$columns_drop = array();
		foreach( $structure as $old_column_def ){
			if( strpos( $old_column_def, '`' ) !== 0 ) continue;

			list( $old_column, ) = explode( ' ', $old_column_def );
			$is_deprecated = TRUE;
			foreach( static::$Q as $new_column_def ){
				if( strpos( $new_column_def, $old_column ) === FALSE ) continue;
				$is_deprecated = FALSE;
				break;
            }

			if( $is_deprecated )
				$columns_drop[] = 'drop colum ' . $old_column;
        }

		if( !empty( $columns_drop ) ){
			$sql = 'alter table `' . $wpdb->prefix . static::$T . '` ' . implode( ', ', $columns_drop) . ';';
			if( !defined( 'ThinkPixel_DEBUG' ) )
			/**
			 * Afraid a little about automatic upgrade
			 */
				$wpdb->query( $sql );
			else
				ThinkPixel_Model::debug( $sql );
        }

		/**
		 * TODO: should check also the keys
		 */
    }

	public static function has_key( $key = null ){
		if( !is_string( $key ) ) return FALSE;
		if( in_array( $key, static::$K ) ) return TRUE;
		return FALSE;
    }

	public static function debug( $message ){
		echo '<!--' . $message . '-->' . "\n";
    }

	public function delete(){
		global $wpdb;
		if( !$this->ID ) throw new Exception();
		$sql = $wpdb->prepare( 'delete from `' . $wpdb->prefix . static::$T . '` where id=%d;', (int) $this->ID );
		if( !defined( 'ThinkPixel_DEBUG' ) )
			$wpdb->query( $sql );
		else
			ThinkPixel_Model::debug( $sql );
    }

	public function crypto(){
		return function_exists( 'bcpowmod') ? bcpowmod( $this->ID, static::$Crypto_GN, static::$Crypto_ZP ) : $this->ID;
    }

	public function __clone(){}

    public function __destruct(){}
}
