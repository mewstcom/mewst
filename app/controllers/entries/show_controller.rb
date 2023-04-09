# typed: true
# frozen_string_literal: true

class Entries::ShowController < ApplicationController
  include Authenticatable
  include Authorizable
  include Localizable

  around_action :set_locale

  sig { returns(T.untyped) }
  def call
    @profile = Profile.only_kept.find_by!(atname: params[:atname])
    @entry = @profile.entries.find(params[:entry_id])
  end
end
