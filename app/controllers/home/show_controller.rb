# typed: true
# frozen_string_literal: true

class Home::ShowController < ApplicationController
  include Authenticatable
  include Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @posts = T.must(current_profile).home_timeline_posts
  end
end
