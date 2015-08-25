require 'sinatra'

set :bind, ARGV[0]

get '/' do
  'hello there world!'
end
