package Digest::SHA1;
use strict; use warnings; use Digest::SHA ();
sub new { my $c = shift; bless { s => Digest::SHA->new(1) }, ref($c)||$c }
sub reset { $_[0]{s} = Digest::SHA->new(1); $_[0] }
sub add { my $s = shift; $s->{s}->add(@_); $s }
sub addfile { my $s = shift; $s->{s}->addfile(@_); $s }
sub hexdigest { $_[0]{s}->hexdigest }   # destructive+reset, like real Digest::SHA1
sub digest    { $_[0]{s}->digest }
1;
