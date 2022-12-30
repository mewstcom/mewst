# typed: strict
# frozen_string_literal: true

class ApplicationPolicy
  extend T::Sig

  sig { returns(Profile) }
  attr_reader :profile

  sig { returns(ApplicationRecord) }
  attr_reader :record

  sig { params(profile: Profile, record: ApplicationRecord).void }
  def initialize(profile, record)
    @profile = profile
    @record = record
  end

  sig { returns(T::Boolean) }
  def index?
    false
  end

  sig { returns(T::Boolean) }
  def show?
    false
  end

  sig { returns(T::Boolean) }
  def create?
    false
  end

  sig { returns(T::Boolean) }
  def new?
    create?
  end

  sig { returns(T::Boolean) }
  def update?
    false
  end

  sig { returns(T::Boolean) }
  def edit?
    update?
  end

  sig { returns(T::Boolean) }
  def destroy?
    false
  end

  class Scope
    extend T::Sig

    T::Sig::WithoutRuntime.sig { params(profile: Profile, scope: ActiveRecord::Relation).void }
    def initialize(profile, scope)
      @profile = profile
      @scope = scope
    end

    sig { void }
    def resolve
      raise NotImplementedError, "You must define #resolve in #{self.class}"
    end

    private

    sig { returns(Profile) }
    attr_reader :profile

    T::Sig::WithoutRuntime.sig { returns(ActiveRecord::Relation) }
    attr_reader :scope
  end
end
