<?php
/**
 * Multi-language support class.
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

class Language {
	const LANGUAGES		= 'thinkpixel_languages';
	const DEFAULT_LANGUAGE	= 'en_US';

	static public $LC = [
		'af' => 'Afrikaans',
		'sq' => 'Albanian',
		'am' => 'Amharic',
		'ar_dz' => 'Arabic - Algeria',
		'ar_bh' => 'Arabic - Bahrain',
		'ar_eg' => 'Arabic - Egypt',
		'ar_iq' => 'Arabic - Iraq',
		'ar_jo' => 'Arabic - Jordan',
		'ar_kw' => 'Arabic - Kuwait',
		'ar_lb' => 'Arabic - Lebanon',
		'ar_ly' => 'Arabic - Libya',
		'ar_ma' => 'Arabic - Morocco',
		'ar_om' => 'Arabic - Oman',
		'ar_qa' => 'Arabic - Qatar',
		'ar_sa' => 'Arabic - Saudi Arabia',
		'ar_sy' => 'Arabic - Syria',
		'ar_tn' => 'Arabic - Tunisia',
		'ar_ae' => 'Arabic - United Arab Emirates',
		'ar_ye' => 'Arabic - Yemen',
		'hy' => 'Armenian',
		'as' => 'Assamese',
		'az_az' => 'Azeri - Cyrillic',
		'az_az' => 'Azeri - Latin',
		'eu' => 'Basque',
		'be' => 'Belarusian',
		'bn' => 'Bengali - Bangladesh',
		'bn' => 'Bengali - India',
		'bs' => 'Bosnian',
		'bg' => 'Bulgarian',
		'my' => 'Burmese',
		'ca' => 'Catalan',
		'zh_cn' => 'Chinese - China',
		'zh_hk' => 'Chinese - Hong Kong SAR',
		'zh_mo' => 'Chinese - Macau SAR',
		'zh_sg' => 'Chinese - Singapore',
		'zh_tw' => 'Chinese - Taiwan',
		'hr' => 'Croatian',
		'cs' => 'Czech',
		'da' => 'Danish',
		'dv' => 'Divehi; Dhivehi; Maldivian',
		'nl_be' => 'Dutch - Belgium',
		'nl_nl' => 'Dutch - Netherlands',
		'en_au' => 'English - Australia',
		'en_bz' => 'English - Belize',
		'en_ca' => 'English - Canada',
		'en_cb' => 'English - Caribbean',
		'en_gb' => 'English - Great Britain',
		'en_in' => 'English - India',
		'en_ie' => 'English - Ireland',
		'en_jm' => 'English - Jamaica',
		'en_nz' => 'English - New Zealand',
		'en_ph' => 'English - Phillippines',
		'en_za' => 'English - Southern Africa',
		'en_tt' => 'English - Trinidad',
		'en_US' => 'English - United States',
		'et' => 'Estonian',
		'fo' => 'Faroese',
		'fa' => 'Farsi - Persian',
		'fi' => 'Finnish',
		'fr_be' => 'French - Belgium',	
		'fr_ca' => 'French - Canada',	
		'fr_fr' => 'French - France',
		'fr_lu' => 'French - Luxembourg',
		'fr_ch' => 'French - Switzerland',
		'mk' => 'FYRO Macedonia',
		'gd_ie' => 'Gaelic - Ireland',
		'gd' => 'Gaelic - Scotland',
		'de_at' => 'German - Austria',
		'de_de' => 'German - Germany',
		'de_li' => 'German - Liechtenstein',
		'de_lu' => 'German - Luxembourg',
		'de_ch' => 'German - Switzerland',
		'el' => 'Greek',
		'gn' => 'Guarani - Paraguay',
		'gu' => 'Gujarati',
		'he' => 'Hebrew',
		'hi' => 'Hindi',
		'hu' => 'Hungarian',
		'is' => 'Icelandic',
		'id' => 'Indonesian',
		'it_it' => 'Italian - Italy',
		'it_ch' => 'Italian - Switzerland',
		'ja' => 'Japanese',
		'kn' => 'Kannada',
		'ks' => 'Kashmiri',
		'kk' => 'Kazakh',
		'km' => 'Khmer',
		'ko' => 'Korean',
		'lo' => 'Lao',
		'la' => 'Latin',
		'lv' => 'Latvian',
		'lt' => 'Lithuanian',
		'ms_bn' => 'Malay - Brunei',
		'ms_my' => 'Malay - Malaysia',
		'ml' => 'Malayalam',
		'mt' => 'Maltese',
		'mi' => 'Maori',
		'mr' => 'Marathi',
		'mn' => 'Mongolian',
		'mn' => 'Mongolian',
		'ne' => 'Nepali',
		'no_no' => 'Norwegian - Bokml',
		'no_no' => 'Norwegian - Nynorsk',
		'or' => 'Oriya',
		'pl' => 'Polish',
		'pt_br' => 'Portuguese - Brazil',
		'pt_pt' => 'Portuguese - Portugal',
		'pa' => 'Punjabi',
		'rm' => 'Raeto-Romance',
		'ro_mo' => 'Romanian - Moldova',
		'ro_RO' => 'Romanian - Romania',
		'ru' => 'Russian',
		'ru_mo' => 'Russian - Moldova',	
		'sa' => 'Sanskrit',
		'sr_sp' => 'Serbian - Cyrillic',
		'sr_sp' => 'Serbian - Latin',
		'tn' => 'Setsuana',
		'sd' => 'Sindhi',
		'si' => 'Sinhala; Sinhalese',
		'sk' => 'Slovak',
		'sl' => 'Slovenian',
		'so' => 'Somali',
		'sb' => 'Sorbian',
		'es_ar' => 'Spanish - Argentina',
		'es_bo' => 'Spanish - Bolivia',
		'es_cl' => 'Spanish - Chile',
		'es_co' => 'Spanish - Colombia',
		'es_cr' => 'Spanish - Costa Rica',
		'es_do' => 'Spanish - Dominican Republic',
		'es_ec' => 'Spanish - Ecuador',
		'es_sv' => 'Spanish - El Salvador',
		'es_gt' => 'Spanish - Guatemala',
		'es_hn' => 'Spanish - Honduras',
		'es_mx' => 'Spanish - Mexico',
		'es_ni' => 'Spanish - Nicaragua',
		'es_pa' => 'Spanish - Panama',
		'es_py' => 'Spanish - Paraguay',
		'es_pe' => 'Spanish - Peru',
		'es_pr' => 'Spanish - Puerto Rico',
		'es_es' => 'Spanish - Spain( Traditional )',
		'es_uy' => 'Spanish - Uruguay',
		'es_ve' => 'Spanish - Venezuela',
		'sw' => 'Swahili',
		'sv_fi' => 'Swedish - Finland',
		'sv_se' => 'Swedish - Sweden',	
		'tg' => 'Tajik',
		'ta' => 'Tamil',
		'tt' => 'Tatar',
		'te' => 'Telugu',
		'th' => 'Thai',
		'bo' => 'Tibetan',
		'ts' => 'Tsonga',
		'tr' => 'Turkish',
		'tk' => 'Turkmen',
		'uk' => 'Ukrainian',
		'ur' => 'Urdu',
		'uz_uz' => 'Uzbek - Cyrillic',
		'uz_uz' => 'Uzbek - Latin',
		'vi' => 'Vietnamese',
		'cy' => 'Welsh',
		'xh' => 'Xhosa',
		'yi' => 'Yiddish',
		'zu' => 'Zulu'
		];

	private $languages;

	public function __construct(){
		if( get_option( self::LANGUAGES ) === FALSE ){
			add_option( self::LANGUAGES, [self::DEFAULT_LANGUAGE => self::$LC[self::DEFAULT_LANGUAGE]] );
			}
		
		$this->languages = get_option( self::LANGUAGES );
		}

	public function set( $key = null, $value = null ){
		if( is_string( $key ) ){
			$slug = self::slug( $key );
			switch( $slug ){
				case 'languages':
					if( !is_array( $value ) || empty( $value ) || !isset( $value[self::DEFAULT_LANGUAGE] )) return FALSE;
					$this->languages = $value;
					update_option( self::LANGUAGES, $this->languages );
					return TRUE;
					break;
				case 'translation':
					$msgid = $value['msgid'];
					$locale = $value['locale'];
					$translation = $value['translation'];

					$po_path = get_template_directory() . DIRECTORY_SEPARATOR . 'lang';
					if( is_null( $locale ))
						$po_file = $po_path . DIRECTORY_SEPARATOR . self::lc( self::DEFAULT_LANGUAGE ) . '.po';
					else
						$po_file = $po_path . DIRECTORY_SEPARATOR . self::lc( $locale ) . '.po';

					if(( $handle = fopen( $po_file, 'r' )) === FALSE ) return FALSE;
					$buffer = '';
					$next_line = FALSE;
					while( ($line = fgets( $handle )) !== FALSE ){
						if( $next_line ){
							$next_line = FALSE;
							$buffer .= 'msgstr "' . $translation . '"' . "\n";
							continue;
							}
						$buffer .= $line;
						if(( $begin = strpos( $line, 'msgid "' . $msgid . '"' )) !== FALSE ){
							$next_line = TRUE;
							continue;
							}
						}
					fclose( $handle );
					file_put_contents( $po_file, $buffer );
					return TRUE;
					break;
				}
			}
		}

	public function get( $key = null, $opts = null ){
		if( is_string( $key ) ){
			$slug = self::slug( $key );
			switch( $slug ){
				case 'languages':
					return $this->languages;
				case 'msgids':
					$msgids = [];
					$po_path = get_template_directory() . DIRECTORY_SEPARATOR . 'lang';
					if( is_null( $opts ))
						$po_file = $po_path . DIRECTORY_SEPARATOR . self::lc( self::DEFAULT_LANGUAGE ) . '.po';
					else
						$po_file = $po_path . DIRECTORY_SEPARATOR . self::lc( $opts ) . '.po';

					if(( $handle = fopen( $po_file, 'r' )) === FALSE ) return FALSE;
					$num = 0;
					while( ($line = fgets( $handle )) !== FALSE ){
						$num ++;
						if(( $begin = strpos( $line, 'msgid "' )) === FALSE ) continue;
						if(( $end = strpos( $line, '"', $begin + 7 )) === FALSE ){
							}
						else {
							$msgid = substr( $line, $begin + 7, $end - $begin - 7 );
							if( !empty( $msgid ))
								$msgids[$num] = $msgid;
							}
						}
					fclose( $handle );
					return $msgids;
					break;
				case 'translation':
					if( empty( $this->languages )) return [];

					$translation = [];

					foreach( $this->languages as $language_locale => $language_name ){
						$po_file = get_template_directory() . DIRECTORY_SEPARATOR . 'lang' . DIRECTORY_SEPARATOR . $language_locale . '.po';

						if(( $handle = fopen( $po_file, 'r' )) === FALSE ) return FALSE;
						$buffer = '';
						$next_line = FALSE;
						while( ($line = fgets( $handle )) !== FALSE ){
							if( $next_line ){
								$next_line = FALSE;
								$translation[$language_locale] = substr( trim( $line ), 8, -1 );
								break;
								}
							$buffer .= $line;
							if(( $begin = strpos( $line, 'msgid "' . $opts . '"' )) !== FALSE ){
								$next_line = TRUE;
								continue;
								}
							}
						fclose( $handle );
						}

					return $translation;
					break;
				}
			}
		}

	public static function scan( $po = '', $path = __DIR__, $mo = FALSE ){
		$stack = [$path];
		$files = [];
		while( !empty( $stack ) ){
			$current = array_pop( $stack );
			if( !is_dir( $current ) ){
				if( is_file( $current ) &&( strrpos( $current, '.php' ) === strlen( $current ) - 4 ))
					$files[] = $current;
				continue;
				}
			if(( $dir = opendir( $current )) === FALSE ) continue;
			while( ($entry = readdir( $dir )) !== FALSE ){
				if( in_array( $entry, ['.', '..'] )) continue;
				$stack[] = $current . DIRECTORY_SEPARATOR . $entry;
				}
			}
		unset( $stack );

		$messages = [];
		foreach( $files as $file ){
			if( $file === __FILE__ ) continue;
			if(( $handle = fopen( $file, 'r' )) === FALSE ) continue;
			$num = 0;
			$base = substr( $file, strlen( $path ) + 1 );
			$msgid = '';
			while( ($line = fgets( $handle )) !== FALSE ){
				$num++;
				if(( ($begin = strpos( $line , '/*T[*/' )) === FALSE ) && empty( $msgid )) continue;
				if( $begin === FALSE ) $begin = -6;
				if(( $end = strpos( $line, '/*]*/', $begin + 6 )) === FALSE )
					$msgid .= substr( $line, $begin + 7 );
				else {
					$msgid .= substr( $line, $begin + 7, $end - $begin - 8 );

					if( isset( $messages[$msgid] ))
						$messages[$msgid][] = $base . ':' . $num;
					else
						$messages[$msgid] = [$base . ':' . $num];

					$msgid = '';
					}
				}
			fclose( $handle );
			}
		unset( $files );

		$handle = fopen( $po, 'w+' );
		if( $handle === FALSE ) return FALSE;
		fwrite( $handle,
'# This file is automagically generated by ThinkPixelLanguage helper class
# AUTHOR: Bogdan DOBRICA @ Asociatia Pro-Youth
#
msgid ""
msgstr ""
"Project-Id-Version: SalesDrive 0.1\n"
"POT-Creation-Date: \n"
"PO-Revision-Date: ' . date( 'Y-m-d H:iO' ) . '\n"
"Last-Translator: Bogdan Dobrica <bdobrica@gmail.com>\n"
"Language-Team: Asociatia Pro-Youth <office@pro-youth.ro>\n"
"MIME-Version: 1.0\n"
"Content-Type: text/plain; charset=utf-8\n"
"Content-Transfer-Encoding: 8bit\n"

' );
		foreach( $messages as $msgid => $files ){
			if( !empty( $files ))
				foreach( $files as $file )
					fwrite( $handle, '#: ' . $file . "\n" );
			fwrite( $handle, 'msgid "' . $msgid . '"' . "\n" );
			if( basename( $po, '.po' ) == self::DEFAULT_LANGUAGE )
				fwrite( $handle, 'msgstr "' . $msgid . '"' . "\n\n" );
			else
				fwrite( $handle, 'msgstr ""' . "\n\n" );
			}
		fclose( $handle );
		unset( $messages );

		if( !$mo ) return TRUE;
		$mo = dirname( $po ) . DIRECTORY_SEPARATOR . basename( $po, '.po' ) . '.mo';
		self::mo( $po, $mo );
		}

	public static function mo( $po, $out ){
		if( !file_exists( $po )) return FALSE;
		$hash = self::_po_parse( $po );
		if( $hash === FALSE ) return FALSE;
		// sort by msgid
		ksort( $hash, SORT_STRING );
		// our mo file data
		$mo = '';
		// header data
		$offsets = [];
		$ids = '';
		$strings = '';

		foreach( $hash as $entry ){
			$id = $entry['msgid'];
			if( isset( $entry['msgid_plural'] ))
				$id .= "\x00" . $entry['msgid_plural'];
			// context is merged into id, separated by EOT( \x04 )
			if( array_key_exists('msgctxt', $entry ))
				$id = $entry['msgctxt'] . "\x04" . $id;
			// plural msgstrs are NUL-separated
			$str = implode("\x00", $entry['msgstr'] );
			// keep track of offsets
			$offsets[] = [strlen($ids ), strlen($id ), strlen($strings ), strlen($str )];
			// plural msgids are not stored( ? )
			$ids .= $id . "\x00";
			$strings .= $str . "\x00";
			}

		// keys start after the header( 7 words ) + index tables( $#hash * 4 words )
		$key_start = 7 * 4 + sizeof($hash ) * 4 * 4;
		// values start right after the keys
		$value_start = $key_start + strlen($ids );
		// first all key offsets, then all value offsets
		$key_offsets = [];
		$value_offsets = [];
		// calculate
		foreach( $offsets as $v ){
			list( $o1, $l1, $o2, $l2 ) = $v;
			$key_offsets[] = $l1;
			$key_offsets[] = $o1 + $key_start;
			$value_offsets[] = $l2;
			$value_offsets[] = $o2 + $value_start;
			}
		$offsets = array_merge( $key_offsets, $value_offsets );

		// write header
		$mo .= pack('Iiiiiii', 0x950412de, // magic number
			0, // version
			sizeof( $hash ), // number of entries in the catalog
			7 * 4, // key index offset
			7 * 4 + sizeof($hash ) * 8, // value index offset,
			0, // hashtable size( unused, thus 0 )
			$key_start // hashtable offset
			);
		// offsets
		foreach( $offsets as $offset )
			$mo .= pack('i', $offset );
		// ids
		$mo .= $ids;
		// strings
		$mo .= $strings;

		return @file_put_contents($out, $mo );
		}

	public static function _po_parse( $in ){
		// read .po file
		$fc = file_get_contents( $in );
		// normalize newlines
		$fc = str_replace( ["\r\n", "\r"], ["\n", "\n"], $fc );
		// results array
		$hash = [];
		// temporary array
		$temp = [];
		// state
		$state = null;
		$fuzzy = FALSE;

		// iterate over lines
		foreach( explode( "\n", $fc ) as $line ){
			$line = trim( $line );
			if( $line === '' )
				continue;

			list( $key, $data ) = explode( ' ', $line, 2 );

			switch( $key ){
				case '#,':	// flag...
					$fuzzy = in_array('fuzzy', preg_split('/,\s*/', $data ));
				case '#':	// translator-comments
				case '#.':	// extracted-comments
				case '#:':	// reference...
				case '#|':	// msgid previous-untranslated-string
						// start a new entry
					if( sizeof( $temp ) && array_key_exists( 'msgid', $temp ) && array_key_exists( 'msgstr', $temp ) ){
						if( !$fuzzy )
							$hash[] = $temp;
						$temp = [];
						$state = null;
						$fuzzy = false;
						}
					break;
				case 'msgctxt':
						// context
				case 'msgid':
						// untranslated-string
				case 'msgid_plural':
						// untranslated-string-plural
					$state = $key;
					$temp[$state] = $data;
					break;
				case 'msgstr':
						// translated-string
					$state = 'msgstr';
					$temp[$state][] = $data;
					break;
				default:
					if( strpos( $key, 'msgstr[' ) !== FALSE ){
						// translated-string-case-n
						$state = 'msgstr';
						$temp[$state][] = $data;
						}
					else {
						// continued lines
						switch( $state ){
							case 'msgctxt':
							case 'msgid':
							case 'msgid_plural':
								$temp[$state] .= "\n" . $line;
								break;
							case 'msgstr':
								$temp[$state][sizeof($temp[$state] ) - 1] .= "\n" . $line;
								break;
							default:
								// parse error
								return FALSE;
							}
						}
					break;
				}
			}
		// add final entry
		if( $state == 'msgstr' )
			$hash[] = $temp;

		// Cleanup data, merge multiline entries, reindex hash for ksort
		$temp = $hash;
		$hash = array();
		foreach( $temp as $entry ){
			foreach( $entry as & $v ){
				$v = self::_po_clean( $v );
				if( $v === FALSE ){
					// parse error
					return FALSE;
					}
				}
			$hash[$entry['msgid']] = $entry;
			}
		return $hash;
		}

	private static function _po_clean( $x ){
		if( is_array( $x ) ){
			foreach( $x as $k => $v ){
				$x[$k] = self::_po_clean( $v );
				}
			}
		else {
			if( $x[0] == '"' )
				$x= substr( $x, 1, -1 );
			$x = str_replace( "\"\n\"", '', $x );
			$x = str_replace( '$', '\\$', $x );
			$x = @eval( "return \"$x\";" );
			}
		return $x;
		}

	public static function lc( $locale ){
		if(( $pos = strpos( $locale, '_' )) === FALSE ) return $locale;
		return strtolower( substr( $locale, 0, $pos )) . strtoupper( substr( $locale, $pos ));
		}

	public static function slug( $key ){
		return trim( preg_replace('/[^a-z0-9]+/', '_', strtolower(trim($key )) ), '_' );
		}
	}
