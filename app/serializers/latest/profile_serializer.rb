# typed: false
# frozen_string_literal: true

class Latest::ProfileSerializer < Latest::ApplicationSerializer
  root_key :profile, :profiles

  attributes :id, :atname, :name, :description, :avatar_url, :viewer_has_followed
end
