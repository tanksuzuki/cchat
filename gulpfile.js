var gulp = require('gulp'),
    sass = require('gulp-sass'),
    notify = require('gulp-notify'),
    autoprefixer = require('gulp-autoprefixer');

var config = {
    sassPath: './scss',
    nodeDir: './node_modules'
}

gulp.task('css', function() {
    return gulp.src(config.sassPath + '/bundle.scss')
        .pipe(sass({
                outputStyle: 'compressed',
                includePaths: [
                    config.sassPath,
                    config.nodeDir + '/bulma',
                ]
            })
            .on("error", notify.onError(function(error) {
                return "Error: " + error.message;
            })))
        .pipe(autoprefixer())
        .pipe(gulp.dest('./public/css'));
});

gulp.task('watch', ['css'], function() {
    gulp.watch(config.sassPath + '/*.scss', ['css']);
});

gulp.task('default', ['css']);
