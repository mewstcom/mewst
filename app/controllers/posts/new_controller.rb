# typed: true
# frozen_string_literal: true

class Posts::NewController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = PostForm.new(with_frame: params[:with_frame])
  end
end
