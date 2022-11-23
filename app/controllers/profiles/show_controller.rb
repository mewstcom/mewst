# typed: true
# frozen_string_literal: true

class Profiles::ShowController < ApplicationController
  include Authenticatable

  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @profile = Profile.find_by!(idname: params[:idname])
  end
end
