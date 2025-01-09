<?php
$header_file = $thinkpixel_theme->get( 'dir', 'request::header' );

$thinkpixel_user = $thinkpixel_theme->get( 'user' );
$user_role = $thinkpixel_theme->get( 'user', 'role' );

if( file_exists( $header_file ))
    include( $header_file );

get_header( );
?>
<div class="container <?php echo !empty( $user_role ) ? 'hube-' . $user_role : ''; ?>">
<?php

if( $user_role == 'admin' ):
    $thinkpixel_theme->render( 'menu' );

?>

<?php $thinkpixel_theme->render( 'header' ); ?>
                <hr />
<?php
    $page_file = $thinkpixel_theme->get( 'dir', 'request::pages' );
    if( file_exists( $page_file )) :
        include( $page_file );
    endif;
?>
        </div>
</div>
<?php else:
    $page_file = $thinkpixel_theme->get( 'dir', 'request::pages' );
    if( file_exists( $page_file )) :
        include( $page_file );
    endif;
endif;
get_footer( );
