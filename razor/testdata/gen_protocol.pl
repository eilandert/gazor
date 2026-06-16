#!/usr/bin/perl
# Generate protocol.tsv: golden vectors for the gazor *protocol* layer, the
# source of truth being real razor. Covers hextobase64 and the full agent
# pipeline (prep_mail -> preproc -> VR4/VR8 signature -> hextobase64).
# Regenerate (run from this directory):
#
#   RAZOR_LIB=/opt/packages/deb/razor/tmp/razor-2.85/lib \
#     perl -Ishim gen_protocol.pl > protocol.tsv
#
# The two preproc chains mirror Razor2::Preproc::Manager exactly: VR4 runs
# deBase64+deQP+deHTML+deNewline, VR8 skips deHTML (Whiplash needs raw URLs).
# (Manager uses the deHTMLxs C module; deHTML.pm is its pure-perl mirror and is
# the documented reference, so we drive the chain with the individual modules.)
use strict;
use warnings;
use lib ($ENV{RAZOR_LIB} || die "set RAZOR_LIB to the razor-2.85/lib path\n");
use Razor2::String qw(prep_mail hextobase64);
use Razor2::Signature::Ephemeral;
use Razor2::Signature::Whiplash;
use Razor2::Preproc::deBase64;
use Razor2::Preproc::deQP;
use Razor2::Preproc::deHTML;
use Razor2::Preproc::deNewline;

# hextobase64 vectors
for my $h (qw(abc 000000 ffffff 123456 a1b2c3d4e5f60718293a4b5c6d7e8f9012345678)) {
    print "B64\t$h\t" . hextobase64($h) . "\n";
}

my $deB = Razor2::Preproc::deBase64->new;
my $deQ = Razor2::Preproc::deQP->new;
my $deH = Razor2::Preproc::deHTML->new;
my $deN = Razor2::Preproc::deNewline->new;

sub preproc_chain {
    my ($text, $do_html) = @_;
    $deB->doit(\$text) if $deB->isit(\$text);
    $deQ->doit(\$text) if $deQ->isit(\$text);
    $deH->doit(\$text) if $do_html && $deH->isit(\$text);
    $deN->doit(\$text);
    my ($hdr, $body) = split /\n\r*\n/, $text, 2;
    return defined($body) ? $body : "";
}

for my $f (sort glob("mail/*.eml")) {
    my $mail = do { local $/; open my $fh, "<", $f or die $!; binmode $fh; <$fh> };
    my ($hdr, @parts) = prep_mail(\$mail, 1, 4 * 1024, 60 * 1024, 15 * 1024, "gazor v2.85", 0);
    my $idx = 0;
    for my $pref (@parts) {
        my $c4 = preproc_chain($$pref, 1);
        my $c8 = preproc_chain($$pref, 0);

        my $e = Razor2::Signature::Ephemeral->new(seed => 7542, separator => "10");
        my $sig4 = hextobase64($e->hexdigest($c4));

        my $w = Razor2::Signature::Whiplash->new;
        my ($wsigs) = $w->whiplash($c8);
        my @s8 = $wsigs ? map { hextobase64($_) } @$wsigs : ();

        print "CLEAN4\t$f\t$idx\t" . unpack("H*", $c4) . "\n";
        print "CLEAN8\t$f\t$idx\t" . unpack("H*", $c8) . "\n";
        print "MAIL\t$f\t$idx\t$sig4\t" . join(",", @s8) . "\n";
        $idx++;
    }
}
