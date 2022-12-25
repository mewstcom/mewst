# typed: true
# frozen_string_literal: true

class Profiles::ShowController < ApplicationController
  include Authenticatable
  include Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @profile = Profile.only_kept.find_by!(atname: params[:atname])
  end
end
