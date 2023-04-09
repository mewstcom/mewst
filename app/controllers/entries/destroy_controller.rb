# typed: true
# frozen_string_literal: true

class Entries::DestroyController < ApplicationController
  include Authenticatable
  include Authorizable
  include Localizable

  around_action :set_locale

  sig { returns(T.untyped) }
  def call
    profile = Profile.only_kept.find_by!(atname: params[:atname])
    entry = profile.entries.find(params[:entry_id])

    authorize(entry, :destroy?)

    current_profile!.delete_entry(entry:)

    flash[:success] = t("messages.posts.deleted")
    redirect_back(fallback_location: root_path)
  end
end
