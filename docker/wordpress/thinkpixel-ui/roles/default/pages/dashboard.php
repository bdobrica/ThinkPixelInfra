<?php
$thinkpixel_language = new \ThinkPixel\Language();
$languages = $thinkpixel_language->get( 'languages' );
?>
<div class="row">
    <div class="col-lg-4 col-md-6 col-sm-12">
        <a href="" class="thinkpixel-logo"><span><?php wp_title(); ?></span></a>
        <div class="thinkpixel-rounded thinkpixel-translucent thinkpixel-padded thinkpixel-center thinkpixel-admin-login">
            <h5><?php \ThinkPixel\Theme::_e( /*T[*/'Admin Login'/*]*/ ); ?></h5>
            <form action="" method="post">
                <label><?php \ThinkPixel\Theme::_e( /*T[*/'Username:'/*]*/ ); ?></label>
                <input class="form-control" name="username" type="text" value="" placeholder="" />
                <label><?php \ThinkPixel\Theme::_e( /*T[*/'Password:'/*]*/ ); ?></label>
                <input class="form-control" name="password" type="password" />
                <label><?php \ThinkPixel\Theme::_e( /*T[*/'Language'/*]*/ ); ?>:</label>
                <?php \ThinkPixel\Theme::inp( 'locale', '', 'select', $languages ); ?>
                <br />
                <br />
                <button class="btn btn-block btn-success"><?php \ThinkPixel\Theme::_e( /*T[*/'Login &raquo;'/*]*/ ); ?></button>
            </form>
            <ul>
                <li><a href="?page=recover"><?php \ThinkPixel\Theme::_e( /*T[*/'Forgot your password?'/*]*/ ); ?></a></li>
                <!--li><a href="?page=register"><?php \ThinkPixel\Theme::_e( /*T[*/'Register a new account?'/*]*/ ); ?></a></li-->
            </ul>
        </div>
    </div>
</div>
