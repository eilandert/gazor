#!/usr/bin/perl
# Generate expected.tsv (golden signature vectors) from REAL razor, the source of
# truth for the gazor parity test. razor's Perl modules are not apt-installable on
# current Debian, so the golden file is committed; regenerate it with:
#
#   RAZOR_LIB=/opt/packages/deb/razor/tmp/razor-2.85/lib \
#     perl -Ishim gen_expected.pl > expected.tsv
#
# (run from this directory; `shim/` provides a Digest::SHA1 compat shim over the
# core Digest::SHA module.)
use strict;
use warnings;
use lib ($ENV{RAZOR_LIB} || die "set RAZOR_LIB to the razor-2.85/lib path\n");
use Razor2::Signature::Ephemeral;
use Razor2::Signature::Whiplash;

my $e = Razor2::Signature::Ephemeral->new();
my $w = Razor2::Signature::Whiplash->new();

for my $f (sort glob("eph/*.txt")) {
    my $c = do { local $/; open my $fh, "<", $f or die $!; binmode $fh; <$fh> };
    print "EPH\t$f\t" . $e->hexdigest($c) . "\n";
}
for my $f (sort glob("whip/*.txt")) {
    my $c = do { local $/; open my $fh, "<", $f or die $!; binmode $fh; <$fh> };
    my ($sigs) = $w->whiplash($c);
    print "WHIP\t$f\t" . ($sigs ? join(",", @$sigs) : "") . "\n";
}
