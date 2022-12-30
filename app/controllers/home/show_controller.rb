# typed: true
# frozen_string_literal: true

class Home::ShowController < ApplicationController
  include Authenticatable
  include Authorizable
  include Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @posts = current_profile!.home_timeline_posts
  end
end
