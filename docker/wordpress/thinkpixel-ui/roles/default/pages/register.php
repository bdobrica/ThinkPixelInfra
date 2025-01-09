<?php ?><div class="row">
    <div class="col-lg-12 col-md-12 col-sm-12">
        <h1><?php wp_title(); ?></h1>
    </div>
</div>
<div class="row">
    <div class="col-lg-4 col-md-6 col-sm-12">
        <div class="thinkpixel-rounded thinkpixel-translucent thinkpixel-padded">
            <h3><?php \ThinkPixel\Theme::_e( /*T[*/'Registration'/*]*/ ); ?></h3>
            <form action="" method="post">
                <label><?php \ThinkPixel\Theme::_e( /*T[*/'Username:'/*]*/ ); ?></label>
                <input class="form-control" name="username" value="" placeholder="" type="text" />
                <label><?php \ThinkPixel\Theme::_e( /*T[*/'First Name:'/*]*/ ); ?></label>
                <input class="form-control" name="username" value="" placeholder="" type="text" />
                <label><?php \ThinkPixel\Theme::_e( /*T[*/'Last Name:'/*]*/ ); ?></label>
                <input class="form-control" name="username" value="" placeholder="" type="text" />
                <label><?php \ThinkPixel\Theme::_e( /*T[*/'E-Mail:'/*]*/ ); ?></label>
                <input class="form-control" name="username" value="" placeholder="" type="text" />
                <label><?php \ThinkPixel\Theme::_e( /*T[*/'Phone:'/*]*/ ); ?></label>
                <input class="form-control" name="username" value="" placeholder="" type="text" />
                <label><?php \ThinkPixel\Theme::_e( /*T[*/'Password:'/*]*/ ); ?></label>
                <input class="form-control" name="username" value="" placeholder="" type="text" />
                <label><?php \ThinkPixel\Theme::_e( /*T[*/'Confirm Password:'/*]*/ ); ?></label>
                <input class="form-control" name="username" value="" placeholder="" type="text" />
                <br />
                <button class="btn btn-block btn-info"><?php \ThinkPixel\Theme::_e( /*T[*/'Register &raquo;'/*]*/ ); ?></button>
            </form>
            <ul>
                <li><a href="?page=recover"><?php \ThinkPixel\Theme::_e( /*T[*/'Forgot your password?'/*]*/ ); ?></a></li>
                <li><a href="?page=dashboard"><?php \ThinkPixel\Theme::_e( /*T[*/'I\'m already registered!'/*]*/ ); ?></a></li>
            </ul>
        </div>
    </div>
</div>
