# typed: true
# frozen_string_literal: true

class Home::ShowController < ApplicationController
  include Authenticatable

  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = PostForm.new
  end
end
